package utils

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
)

func MD5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func SHA1(text string) string {
	hasher := sha1.New()
	hasher.Write([]byte(text))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func HMAC(key, msg, signMethod string) []byte {
	var hashFunc func() hash.Hash
	switch signMethod {
	case "sha1":
		hashFunc = sha1.New
	default:
		hashFunc = sha1.New
	}
	h := hmac.New(hashFunc, []byte(key))
	h.Write([]byte(msg))
	return h.Sum(nil)
}

func BASE64(sign []byte, orig string) string {
	m := append(sign, []byte(orig)...)
	sEnc := base64.StdEncoding.EncodeToString(m)
	return sEnc
}

func BASE64Decode(str string) string {
	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return ""
	}
	return string(decoded)
}
