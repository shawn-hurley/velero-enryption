package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
)

// NewStreamEncrypter creates a new stream encrypter
func NewStreamEncrypter(encKey, macKey []byte, plainText io.Reader) (*StreamEncrypter, error) {
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, block.BlockSize())
	_, err = rand.Read(iv)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, iv)
	mac := hmac.New(sha256.New, macKey)
	return &StreamEncrypter{
		Source: plainText,
		Block:  block,
		Stream: stream,
		Mac:    mac,
		IV:     iv,
	}, nil
}

// NewStreamDecrypter creates a new stream decrypter
func NewStreamDecrypter(encKey, macKey []byte, cipherText io.ReadCloser) (*StreamDecrypter, error) {
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, macKey)
	return &StreamDecrypter{
		Source: cipherText,
		Block:  block,
		Mac:    mac,
	}, nil
}

// StreamEncrypter is an encrypter for a stream of data with authentication
type StreamEncrypter struct {
	Source io.Reader
	Block  cipher.Block
	Stream cipher.Stream
	Mac    hash.Hash
	IV     []byte
}

// StreamDecrypter is a decrypter for a stream of data with authentication
type StreamDecrypter struct {
	Source io.ReadCloser
	Block  cipher.Block
	Mac    hash.Hash
}

// Populate IV and Hash
func (s *StreamEncrypter) EncryptData(outFile io.Writer) error {
	buf := make([]byte, 4096)

	for {
		n, err := s.Source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}

		outBuf := make([]byte, n)
		s.Stream.XORKeyStream(outBuf, buf[:n])
		s.Mac.Write(outBuf)
		outFile.Write(outBuf)

		if err == io.EOF {
			break
		}
	}

	outFile.Write(s.IV)
	s.Mac.Write(s.IV)
	outFile.Write(s.Mac.Sum(nil))
	return nil
}

// add meta to the output

// Pull all the data into memory, to get the hash and iv,
func (s *StreamDecrypter) DecryptData(outFile io.Writer) error {
	buf := &bytes.Buffer{}
	io.Copy(buf, s.Source)
	defer s.Source.Close()

	dataBytes := buf.Bytes()
	if len(dataBytes) < (32 + s.Block.BlockSize()) {
		return fmt.Errorf("unable to decrypt as data does not contain hash and IV")
	}
	hash := dataBytes[len(dataBytes)-32:]
	s.Mac.Write(dataBytes[:len(dataBytes)-32])
	if !bytes.Equal(hash, s.Mac.Sum(nil)) {
		return fmt.Errorf("invalid data")
	}

	iv := dataBytes[len(dataBytes)-(32+s.Block.BlockSize()) : len(dataBytes)-32]
	stream := cipher.NewCTR(s.Block, iv)

	dataBuf := bytes.NewBuffer(dataBytes[:len(dataBytes)-(32+s.Block.BlockSize())])
	interBuf := make([]byte, 4096)
	for {
		n, err := dataBuf.Read(interBuf)
		if err != nil && err != io.EOF {
			return err
		}

		outBuf := make([]byte, n)
		stream.XORKeyStream(outBuf, interBuf[:n])
		outFile.Write(outBuf)

		if err == io.EOF {
			break
		}
	}

	return nil

}
