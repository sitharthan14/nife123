package decode

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"os"
)

// var PwdKey = "linkbook1qaz*WSX"

//PKCS7 padding mode
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	//The function of the Repeat() function is to copy the padding of the slice []byte{byte(padding)}, and then merge it into a new byte slice to return
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

//The reverse operation of filling, delete the filling string
func PKCS7UnPadding1(origData []byte) ([]byte, error) {
	//Get data length
	length := len(origData)
	if length == 0 {
		return nil, errors.New("Encrypted string error!")
	} else {
		//Get the fill string length
		unpadding := int(origData[length-1])
		//Intercept the slice, delete the padding bytes, and return the plaintext
		return origData[:(length - unpadding)], nil
	}
}

//Encryption
func AesEcrypt(origData []byte, key []byte) ([]byte, error) {
	//Create an instance of the encryption algorithm
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//Get the size of the block
	blockSize := block.BlockSize()
	//Padded the data so that the data length meets the demand
	origData = PKCS7Padding(origData, blockSize)
	//CBC encryption mode in AES encryption method
	blocMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(origData))
	//Perform encryption
	blocMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

//Realize decryption
func AesDeCrypt(cypted []byte, key []byte) (string, error) {
	//Create an instance of the encryption algorithm
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	//Get the block size
	blockSize := block.BlockSize()
	//Create an encrypted client instance
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(cypted))
	//This function can also be used to decrypt
	blockMode.CryptBlocks(origData, cypted)
	//Remove the fill string
	origData, err = PKCS7UnPadding1(origData)
	if err != nil {
		return "", err
	}
	return string(origData), err
}

//Encrypted base64
func EnPwdCode(pwdStr string) string {
	pwd := []byte(pwdStr)
	var PwdKey = os.Getenv("ENCRYPTION_KEY")
	result, err := AesEcrypt(pwd, []byte(PwdKey))
	if err != nil {
		return ""
	}
	return hex.EncodeToString(result)
}

//Decrypt
func DePwdCode(pwd string) string {
	temp, _ := hex.DecodeString(pwd)
	//Perform AES decryption
	var PwdKey = os.Getenv("ENCRYPTION_KEY")
	res, _ := AesDeCrypt(temp, []byte(PwdKey))
	return res
}
