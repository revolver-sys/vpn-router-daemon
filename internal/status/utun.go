package status

import (
	"github.com/revolver-sys/vpn-router-daemon/internal/utun"
)

func ListUTUN() ([]string, error) {
	return utun.List()
}
