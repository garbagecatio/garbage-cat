package store

import (
	"os"
)

var filename = "./secret.txt"

func StoreExists() bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func StoreMnemonicToFile(mnemonic string) error {
	err := os.WriteFile(filename, []byte(mnemonic), 0600)
	if err != nil {
		return err
	}
	return nil
}

func ReadMnemonicFromFile() (string, error) {
	// Reading the mnemonic back (for demonstration purposes)
	readMnemonic, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return string(readMnemonic), nil
}
