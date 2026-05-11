package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

func Health(e *core.RequestEvent) error {
	return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
