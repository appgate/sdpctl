package auth

// OpenIDConnectHTML is the signin complete view.
// same layout as the client uses.
const OpenIDConnectHTML = `
<html><head>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Appgate SDP</title>
<style>
    body {
        font-family: 'Roboto', 'Helvetica Neue', Helvetica, Arial, sans-serif;
    }

    h1 {
        font-size: 20px;
        color: dimgrey;
        font-weight: 600;
        margin-top: 100px;
        margin-bottom: 5px;
        padding: 0;
    }

    h2 {
        font-size: 18px;
        color: grey;
        font-weight: 400;
        margin-top: 0;
        margin-bottom: 60px;
        padding: 0;
    }

    a {
        font-size: 20px;
        margin-top: 20px;
        background: #0799e4;
        color: white;
        text-decoration: none;
        display: inline-flex;
        height: 60px;
        width: 249px;
        justify-content: center;
        align-items: center;
    }

    .center {
        position: relative;
        text-align: center;
    }

    .illustration {
        display: block;
    }

    svg {
        width: 260px;
        height: 197px;
    }
</style></head>

<body>
    <div class="center">

        <h1>Successfully authenticated with external provider</h1>
        <h2>You may close this window</h2>

        <div class="illustration">
            <!--?xml version="1.0" encoding="UTF-8"?-->
            <svg viewBox="0 0 149 109" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
                <!-- Generator: Sketch 42 (36781) - http://www.bohemiancoding.com/sketch -->
                <defs>
                    <rect id="path-1" x="0" y="0" width="149" height="109" rx="12"></rect>
                    <mask id="mask-2" maskContentUnits="userSpaceOnUse" maskUnits="objectBoundingBox" x="0" y="0" width="149" height="109" fill="white">
                        <use xlink:href="#path-1"></use>
                    </mask>
                </defs>
                <g id="Page-2" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
                    <g id="Mobile-Portrait-Copy-4" transform="translate(-118.000000, -243.000000)">
                        <g id="saml-close" transform="translate(118.000000, 243.000000)">
                            <path d="M72.6966392,69.7703176 L112.298449,30.1685077 C113.858431,28.6085259 116.390512,28.6113768 117.950929,30.1717934 L126.437299,38.6581638 C127.999095,40.2199602 128.004354,42.7468741 126.440585,44.3106433 L75.5212361,95.229992 C74.7412452,96.0099829 73.7182295,96.3992656 72.6951594,96.3984985 C71.6714336,96.3957099 70.6459034,96.0038531 69.8631282,95.2210779 L40.1748111,65.5327608 C38.6099062,63.9678559 38.6087661,61.4317838 40.1691827,59.8713672 L48.6555531,51.3849968 C50.2173496,49.8232004 52.752175,49.8258535 54.3169468,51.3906252 L72.6966392,69.7703176 Z" id="Combined-Shape" fill="#039BE5"></path>
                            <g id="browser" opacity="0.304064764">
                                <use id="Rectangle-3-Copy-2" stroke="#039BE5" mask="url(#mask-2)" stroke-width="6" xlink:href="#path-1"></use>
                                <path d="M149,27.4552462 L149,11.9982389 C149,5.366317 143.625543,0 136.995815,0 L12.0041848,0 C5.36894649,0 0,5.37179455 0,11.9982389 L0,27.4552462 L0,19.1167804 L149,19.1167804 L149,27.4552462 Z" id="Combined-Shape-Copy" fill="#039BE5"></path>
                                <circle id="Oval" fill="#EEEEEE" cx="134.5" cy="10.5" r="3.5"></circle>
                                <circle id="Oval-Copy" fill="#EEEEEE" cx="123.5" cy="10.5" r="3.5"></circle>
                                <circle id="Oval-Copy-2" fill="#EEEEEE" cx="112.5" cy="10.5" r="3.5"></circle>
                                <rect id="Rectangle-3" fill="#039BE5" x="33" y="45" width="67" height="7"></rect>
                                <rect id="Rectangle-3-Copy-4" fill="#039BE5" x="33" y="38" width="28" height="4"></rect>
                                <rect id="Rectangle-3-Copy-5" fill="#039BE5" x="33" y="60" width="28" height="4"></rect>
                                <rect id="Rectangle-3-Copy-3" fill="#039BE5" x="33" y="67" width="67" height="7"></rect>
                                <rect id="Rectangle-3-Copy-6" fill="#039BE5" x="107" y="90" width="28" height="7"></rect>
                            </g>
                        </g>
                    </g>
                </g>
            </svg>
        </div>
        <a id="closeButton" href="javascript:window.open('','_self').close();" style="display: none;">CLOSE</a>
    </div>
<script>
    // > for security reasons, scripts are no longer allowed to close windows they didn't open. (Firefox 46.0.1: scripts can not close windows, they had not opened)
    if (!window.opener && window.history.length > 1) {
        document.getElementById('closeButton').style.display = 'none';
    }
    else {

    }
</script>
</body></html>
`
