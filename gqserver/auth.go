package gqserver

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"log"
	"sync"
)

func decrypt(iv []byte, key []byte, ciphertext []byte) []byte {
	ret := make([]byte, len(ciphertext))
	copy(ret, ciphertext) // Because XORKeyStream is inplace, but we don't want the input to be changed
	block, _ := aes.NewCipher(key)
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ret, ret)
	// ret is now plaintext
	return ret
}

// IsSS checks if a ClientHello belongs to shadowsocks
func IsSS(input *ClientHello, sta *State) bool {
	ticket := input.extensions[[2]byte{0x00, 0x23}]
	if len(ticket) != 192 {
		return false
	}

	var random [32]byte
	copy(random[:], input.random)

	var mutex = &sync.Mutex{}

	mutex.Lock()
	used := sta.UsedRandom[random]
	mutex.Unlock()

	if used != 0 {
		log.Println("Replay! Duplicate random")
		return false
	}

	mutex.Lock()
	sta.UsedRandom[random] = int(sta.Now().Unix())
	mutex.Unlock()

	h := sha256.New()
	t := int(sta.Now().Unix()) / sta.TicketTimeHint
	h.Write([]byte(fmt.Sprintf("%v", t) + sta.Key))
	goal := h.Sum(nil)[0:16]
	plaintext := decrypt(input.random[0:16], sta.AESKey, input.random[16:])
	return bytes.Equal(plaintext, goal)

}