package main

import (
	"bufio"
	"fmt"
	"os"
)

type manualDnsResponder struct {
	records map[string]string
}

func newManualDnsResponder() *manualDnsResponder {
	return &manualDnsResponder{records: make(map[string]string)}
}

func (r *manualDnsResponder) Cleanup() {
	for fqdn := range r.records {
		fmt.Fprintf(os.Stderr, "Delete DNS record: \"%s\"\n", fqdn)
	}
	r.records = make(map[string]string)
}

func (r *manualDnsResponder) Respond(key, value string) error {
	if r.records[key] == value {
		return nil
	}
	fmt.Fprintf(os.Stderr,
		"Add TXT record for: \"%s\", value: \"%s\" and then press ENTER\n",
		key, value)
	reader := bufio.NewReader(os.Stdin)
	if _, err := reader.ReadString('\n'); err != nil {
		return fmt.Errorf("error reading input: %s", err)
	}
	r.records[key] = value
	return nil
}
