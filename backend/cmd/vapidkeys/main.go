// Command vapidkeys prints a fresh VAPID key pair for Web Push in env-file format.
//
//	go run ./cmd/vapidkeys [-subject mailto:admin@example.org]
//
// Paste the lines into the API's env file (deploy/api.env.example). Keep the private
// key secret; changing the key later invalidates all existing browser subscriptions.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
)

func main() {
	subject := flag.String("subject", "mailto:admin@example.org", "contact for the push services (mailto: or https: URI)")
	flag.Parse()
	pub, priv, err := push.GenerateVAPIDKeys()
	if err != nil {
		fmt.Fprintln(os.Stderr, "vapidkeys:", err)
		os.Exit(1)
	}
	fmt.Printf("REITERHOF_VAPID_PUBLIC_KEY=%s\n", pub)
	fmt.Printf("REITERHOF_VAPID_PRIVATE_KEY=%s\n", priv)
	fmt.Printf("REITERHOF_VAPID_SUBJECT=%s\n", *subject)
}
