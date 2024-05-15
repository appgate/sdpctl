package files

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"sync"

	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"
)

type QueueItem struct {
	File       *os.File
	size       int64
	RemoteName string
}

type FileManager struct {
	API      *appliancepkg.Appliance
	queue    []QueueItem
	Progress *tui.Progress
}

func NewFileManager(api *appliancepkg.Appliance, progress *tui.Progress, files ...QueueItem) *FileManager {
	m := &FileManager{
		API:      api,
		Progress: progress,
	}
	for _, f := range files {
		m.AddToQueue(f.File, f.RemoteName)
	}
	return m
}

func (f *FileManager) AddToQueue(file *os.File, remoteName string) error {
	file, err := os.Open(file.Name())
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	size := info.Size()

	// make sure file is closed when adding to queue
	// we ignore the error if the file is already closed
	file.Close()

	f.queue = append(f.queue, QueueItem{
		File:       file,
		size:       size,
		RemoteName: remoteName,
	})
	return nil
}

func (f *FileManager) TotalQueueSize() int64 {
	var total int64
	for _, q := range f.queue {
		total += q.size
	}
	return total
}

func (f *FileManager) QueueLen() int {
	return len(f.queue)
}

func (f *FileManager) FileNames() []string {
	fn := make([]string, 0, f.QueueLen())
	for _, file := range f.queue {
		fn = append(fn, file.File.Name())
	}
	return fn
}

func (f *FileManager) WorkQueue(ctx context.Context) error {
	var errs *multierror.Error
	ec := make(chan error)
	var wg sync.WaitGroup
	msg := fmt.Sprintf("Uploading %d file(s)", f.QueueLen())
	if f.Progress != nil {
		f.Progress.WriteLine(msg, "\n")
	}
	log.WithFields(log.Fields{
		"files": strings.Join(f.FileNames(), ","),
	}).Info(msg)
	for _, q := range f.queue {
		wg.Add(1)
		go func(wg *sync.WaitGroup, ec chan error, q QueueItem) {
			defer wg.Done()
			ec <- f.Upload(ctx, q)
		}(&wg, ec, q)
	}

	go func(wg *sync.WaitGroup, ec chan error) {
		wg.Wait()
		close(ec)
	}(&wg, ec)

	for err := range ec {
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	if f.Progress != nil {
		f.Progress.Wait()
	}
	return errs.ErrorOrNil()
}

func (f *FileManager) Upload(ctx context.Context, q QueueItem) error {
	var err error
	q.File, err = os.Open(q.File.Name())
	if err != nil {
		return err
	}
	fileInfo, err := q.File.Stat()
	if err != nil {
		return err
	}
	name := fileInfo.Name()
	size := q.size
	if size <= 0 {
		size = fileInfo.Size()
	}
	uploadName := name
	if len(q.RemoteName) > 0 {
		uploadName = q.RemoteName
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	go func() {
		defer pw.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("file", uploadName)
		if err != nil {
			log.Warnf("multipart form err %s", err)
			return
		}

		_, err = io.Copy(part, q.File)
		if err != nil {
			log.Warnf("copy err %s", err)
		}
	}()

	headers := map[string]string{
		"Content-Type":        writer.FormDataContentType(),
		"Content-Disposition": fmt.Sprintf("attachment; filename=%s", uploadName),
	}

	var reader io.ReadCloser
	reader = pr
	defer reader.Close()
	var t *tui.Tracker
	endMsg := "uploaded"
	if f.Progress != nil {
		progressString := name
		if len(q.RemoteName) > 0 && q.RemoteName != name {
			progressString = fmt.Sprintf("%s -> %s", progressString, q.RemoteName)
		}
		reader, t = f.Progress.FileUploadProgress(progressString, endMsg, size, pr)
		go t.Watch([]string{endMsg}, []string{appliancepkg.FileFailed})
	}
	if err := f.API.UploadFile(ctx, reader, headers); err != nil {
		if t != nil {
			t.Fail(err.Error())
		}
		return err
	}
	return func(ctx context.Context, api *appliancepkg.Appliance) error {
		return backoff.Retry(func() error {
			v, err := api.FileStatus(ctx, name)
			if err != nil {
				if t != nil {
					t.Fail(err.Error())
				}
				return backoff.Permanent(err)
			}
			status := v.GetStatus()
			if t != nil {
				t.Update(status)
			}
			if status == appliancepkg.FileReady {
				if t != nil {
					t.Update(endMsg)
				}
				return nil
			}
			if status == appliancepkg.FileFailed {
				return backoff.Permanent(fmt.Errorf("%s failed: %q", name, err))
			}
			return fmt.Errorf("file not ready")
		}, backoff.NewExponentialBackOff())
	}(ctx, f.API)
}
