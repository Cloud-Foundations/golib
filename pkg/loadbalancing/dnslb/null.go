package dnslb

type nullType struct{}

var nullInterface = &nullType{}

func (nullType) Destroy(ips map[string]struct{}) error {
	return nil
}

func (nullType) Filter(ips map[string]struct{}) (map[string]struct{}, error) {
	return ips, nil
}
