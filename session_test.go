package whatsapp

import "testing"

func TestTsUpdateResponse(t *testing.T) {
	if !isUpdateResponse(`["Cmd",{"type":"update"}]`) {
		t.Error("Update response not detected")
	}
}
