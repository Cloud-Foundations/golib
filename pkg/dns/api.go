/*
Package dns defines generic interfaces for managing DNS records.
*/
package dns

import "time"

// RecordDeleter defines a DNS record deleter.
type RecordDeleter interface {
	DeleteRecords(fqdn, recType string) error
}

// RecordDeleteWriter defines a DNS record deleter/writer.
type RecordDeleteWriter interface {
	RecordDeleter
	RecordWriter
}

// RecordReader defines a DNS record reader.
type RecordReader interface {
	ReadRecords(fqdn, recType string) ([]string, time.Duration, error)
}

// RecordWriter defines a DNS record writer.
type RecordWriter interface {
	WriteRecords(fqdn, recType string, recs []string, ttl time.Duration,
		wait bool) error
}

// RecordManager defines a DNS record manager.
type RecordManager interface {
	RecordDeleter
	RecordReader
	RecordWriter
}
