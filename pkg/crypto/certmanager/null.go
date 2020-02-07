package certmanager

type nullLocker struct{}

func (nullLocker) GetLostChannel() <-chan error {
	return nil
}

func (nullLocker) Lock() error {
	return nil
}

func (nullLocker) Unlock() error {
	return nil
}
