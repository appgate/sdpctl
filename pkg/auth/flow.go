package auth

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"runtime"
)

const mfaHTML = `
<!doctype html>
<meta charset="utf-8">
<title>Appgate SDP OTP Initialization</title>
<style type="text/css">
body {
  color: #1B1F23;
  background: #e6e6e6;
  font-size: 14px;
  font-family: -apple-system, "Segoe UI", Helvetica, Arial, sans-serif;
  line-height: 1.5;
  max-width: 620px;
  margin: 28px auto;
  text-align: center;
}

h1 {
  font-size: 24px;
  margin-bottom: 0;
}

p {
  margin-top: 0;
}

.box {
  border: 1px solid #E1E4E8;
  background: white;
  padding: 24px;
  margin: 28px;
}
</style>
<body>
  <div class="box">
	<h1>1. Download app</h1>
	<p>For Android and iOS: Google Authenticator</p>

	<h1> 2. Scan QR code </h1>
	<img id="qr-image" src="data:image/jpg;base64,{{ .Barcode }}" alt="QR code">
	<p>Scan the image above using the app.
	If you can’t use the code, enter {{ .Secret }}</p>
  </div>
</body>
`

func Openbrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}

func BarcodeHTMLfile(barcode, secret string) (*os.File, error) {
	t := template.Must(template.New("").Parse(mfaHTML))
	var tpl bytes.Buffer
	type stub struct {
		Barcode, Secret string
	}

	data := stub{
		Barcode: barcode,
		Secret:  secret,
	}
	if err := t.Execute(&tpl, data); err != nil {
		return nil, err
	}
	file, err := os.CreateTemp("", "appgate_mfa_*.html")
	if err != nil {
		return file, err
	}
	_, err = file.WriteString(tpl.String())
	if err != nil {
		return file, err
	}

	return file, nil
}
