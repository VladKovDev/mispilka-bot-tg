package crypto

type KeyStore struct{
	Encryptors map[int]Encryptor
	Current int
}

func NewAESKeyStore(current int, keys map[int][]byte) (*KeyStore, error) {
	ks := &KeyStore{
		Encryptors: make(map[int]Encryptor),
		Current: current,
	}

	for ver, key := range keys {
		enc, err := NewAESEncryptor(key)
		if err != nil {
			return nil, err
		}
		ks.Encryptors[ver] = enc
	}

	return ks, nil
}