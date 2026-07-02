package overlay

import (
	"strings"
	"text/template"
)

type sefData struct {
	SiteName   string
	ConnectURL string
}

const sefTemplate = `[extension_name]
{{.SiteName}} Overlay
[extension_info]
Connects SAMMI to {{.SiteName}}. Site events (likes, follows, comments, theory votes) fire the 'overlay_event' extension trigger so you can show overlays on stream. Your personal connection token is baked into this file: keep it private and re-download if you reset it.
[extension_version]
1.0.0
[insert_external]
<script>
    let umi_ws = null;
    let umi_should_reconnect = false;
    const UMI_WS_URL = "{{.ConnectURL}}";
</script>
<div class="row">
  <div class="col-12 col-md-8 col-lg-6 mx-auto">
    <div class="card bg-dark text-white">
      <div class="card-header">
        <h3>{{.SiteName}} Overlay</h3>
      </div>
      <div class="card-body">
        <p>Run <strong>Overlay: Connect</strong> on deck load, then catch the <strong>overlay_event</strong> extension trigger on a button and read its data with Trigger Pull Data.</p>
      </div>
    </div>
  </div>
</div>
[insert_command]
SAMMI.extCommand('Overlay: Connect', 3355443, 52, {});
SAMMI.extCommand('Overlay: Disconnect', 3355443, 52, {});
[insert_hook]
[insert_script]
function Umineko_Connect() {
    if (umi_ws && (umi_ws.readyState === WebSocket.OPEN || umi_ws.readyState === WebSocket.CONNECTING)) {
        return;
    }
    umi_should_reconnect = true;
    umi_ws = new WebSocket(UMI_WS_URL);
    umi_ws.onopen = function () {
        SAMMI.triggerExt('overlay_connected', {});
    };
    umi_ws.onmessage = function (event) {
        let data;
        try {
            data = JSON.parse(event.data);
        } catch (e) {
            data = { raw: event.data };
        }
        SAMMI.triggerExt('overlay_event', data);
    };
    umi_ws.onclose = function () {
        SAMMI.triggerExt('overlay_disconnected', {});
        if (umi_should_reconnect) {
            setTimeout(Umineko_Connect, 5000);
        }
    };
    umi_ws.onerror = function () {
        if (umi_ws) {
            umi_ws.close();
        }
    };
}

function Umineko_Disconnect() {
    umi_should_reconnect = false;
    if (umi_ws) {
        umi_ws.close();
    }
}

function Umineko_Main() {
    sammiclient.on('Overlay: Connect', function () {
        Umineko_Connect();
    });
    sammiclient.on('Overlay: Disconnect', function () {
        Umineko_Disconnect();
    });
}

Umineko_Main();
`

var sefTmpl = template.Must(template.New("sef").Parse(sefTemplate))

func renderSEF(connectURL, siteName string) (string, error) {
	var b strings.Builder
	if err := sefTmpl.Execute(&b, sefData{SiteName: siteName, ConnectURL: connectURL}); err != nil {
		return "", err
	}
	return b.String(), nil
}
