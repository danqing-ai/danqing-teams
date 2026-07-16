package v1

import _ "embed"

//go:embed inspect_inject.js
var dqInspectJS []byte

// dqInspectScript is injected into HTML responses so the browser tab can select elements.
var dqInspectScript = "<script>" + string(dqInspectJS) + "</script>"
