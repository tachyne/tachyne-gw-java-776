// Command gw runs the tachyne Java gateway pinned to protocol 776 ("26.2").
// The entire gateway — front door and session pipeline — lives in
// tachyne-common/gwsession; this binary is only the version pinning +
// environment wiring.
//
// Configuration is env-first (Kubernetes style):
//
//	TACHYNE_LISTEN   client-facing listen address       (default ":25565")
//	TACHYNE_BACKEND  world pod attach address           ("" = under-construction farewell)
//	TACHYNE_MOTD     server-list description            (default derived from version)
//	POD_NAME         StatefulSet pod name; the trailing ordinal becomes the
//	                 gateway's SID for inter-server comms (default 0)
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tachyne/tachyne-common/access"
	"github.com/tachyne/tachyne-common/gwsession"
)

// Protocol pinning for this gateway build: exactly one client version; the
// translation chain rewrites canonical 770 to it at the client edge.
const (
	Protocol    = 776    // Minecraft Java network protocol this gateway serves
	VersionName = "26.2" // human-readable release name for that protocol
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	s := &gwsession.Server{
		Listen:       envOr("TACHYNE_LISTEN", ":25565"),
		Backend:      os.Getenv("TACHYNE_BACKEND"),
		WorldPattern: envOr("TACHYNE_WORLD_PATTERN", "tachyne-world-%d.tachyne-world-hl.tachyne.svc.cluster.local:25500"),
		AttachToken:  os.Getenv("TACHYNE_ATTACH_TOKEN"),
		MOTD:         envOr("TACHYNE_MOTD", "tachyne — Minecraft "+VersionName+" gateway"),
		SID:          ordinal(os.Getenv("POD_NAME")),
		Name:         "gw-java-776",
		VersionName:  VersionName,
		Proto:        Protocol,
		MinProto:     Protocol,
		MaxProto:     Protocol,
	}
	if url := os.Getenv("TACHYNE_ACCESS_URL"); url != "" {
		s.Access = access.New(url, os.Getenv("TACHYNE_ACCESS_TOKEN"), 30*time.Second)
		log.Printf("access control via %s (fail closed)", url)
	} else {
		log.Print("WARNING: TACHYNE_ACCESS_URL unset — running OPEN (no access control)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("tachyne-gw-java sid=%d proto=%d (%s) listening on %s", s.SID, Protocol, VersionName, s.Listen)
	if err := s.Run(ctx); err != nil {
		log.Fatal(err)
	}
	log.Print("shut down")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ordinal extracts the StatefulSet ordinal from a pod name
// ("tachyne-gw-java-776-3" → 3). Anything unparseable is SID 0.
func ordinal(podName string) int {
	i := strings.LastIndexByte(podName, '-')
	if i < 0 {
		return 0
	}
	n, err := strconv.Atoi(podName[i+1:])
	if err != nil || n < 0 {
		return 0
	}
	return n
}
