package doubao

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
)

func BuildVolcesSubmitResponse(publicTaskID string) ([]byte, error) {
	return common.Marshal(map[string]any{
		"id": publicTaskID,
	})
}

func BuildVolcesFetchResponse(publicTaskID string, rawBody []byte) ([]byte, error) {
	if len(rawBody) == 0 {
		return nil, fmt.Errorf("empty fetch response body")
	}
	var payload map[string]any
	if err := common.Unmarshal(rawBody, &payload); err != nil {
		return nil, err
	}
	payload["id"] = publicTaskID
	return common.Marshal(payload)
}
