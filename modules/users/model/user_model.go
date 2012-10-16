package user_model

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	iface "github.com/opesun/chill/frame/interfaces"
	"github.com/opesun/chill/frame/misc/convert"
	"github.com/opesun/slugify"
	"io"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strings"
)

const (
	block_size = 16 // For encryption and decryption.
)

type m map[string]interface{}

// Finds a user by id.
func FindUser(a iface.Filter, id bson.ObjectId) (map[string]interface{}, error) {
	q := m{"_id": id}
	v, err := a.AddQuery(q).FindOne()
	if err != nil {
		return nil, err
	}
	delete(v, "password")
	return v, nil
}

// Finds he user by name password equality.
func namePass(a iface.Filter, name, encoded_pass string) (map[string]interface{}, error) {
	q := bson.M{
		"name": name,
		"password": encoded_pass,
	}
	doc, err := a.AddQuery(q).FindOne()
	if err != nil {
		return nil, err
	}
	return convert.Clean(doc).(map[string]interface{}), nil
}

func FindLogin(a iface.Filter, name, password string) (map[string]interface{}, bson.ObjectId, error) {
	pass := hashPass(password)
	user, err := namePass(a, name, pass)
	if err != nil {
		return nil, "", err
	}
	return user, user["_id"].(bson.ObjectId), nil
}

// Sets a cookie to w named "user" with a value of the encoded user_id.
// Admins, guests, registered users, everyone logs in with this.
func Login(w http.ResponseWriter, user_id bson.ObjectId, block_key []byte) error {
	id_b, err := encryptStr(block_key, user_id.Hex())
	if err != nil {
		return err
	}
	encoded_id := string(id_b)
	c := &http.Cookie{
		Name:   "user",
		Value:  encoded_id,
		MaxAge: 3600000,
		Path:   "/",
	}
	http.SetCookie(w, c)
	return nil
}

// When no user cookie is found, or there was a problem during building the user,
// we proceed with an empty user.
func EmptyUser() map[string]interface{} {
	user := make(map[string]interface{})
	user["level"] = 0
	return user
}

func parseAcceptLanguage(l string) []string {
	ret := []string{}
	sl := strings.Split(l, ",")
	c := map[string]struct{}{}
	for _, v := range sl {
		lang := string(strings.Split(v, ";")[0][:2])
		_, has := c[lang]
		if !has {
			c[lang] = struct{}{}
			ret = append(ret, lang)
		}
	}
	return ret
}

// Creates a list of 2 char language abbreviations (for example: []string{"en", "de", "hu"}) out of the value of http header "Accept-Language".
func ParseAcceptLanguage(l string) (ret []string) {
	defer func(){
		r := recover()
		if r != nil {
			ret = []string{"en"}
		}
	}()
	ret = parseAcceptLanguage(l)
	return
}

// Decrypts a string with block_key.
// Also decodes val from base64.
// This is put here as a separate function (has no public Encrypt pair) to be able to separate the decryption of the
// cookie into a user_id (see DecryptId)
func Decrypt(val string, block_key []byte) (string, error) {
	block_key = block_key[:block_size]
	decr_id_b, err := decryptStr(block_key, val)
	if err != nil {
		return "", err
	}
	return string(decr_id_b), nil
}

// cookieval is encrypted
// Converts an encoded string (a cookie) into an ObjectId.
func DecryptId(cookieval string, block_key []byte) (bson.ObjectId, error) {
	str, err := Decrypt(cookieval, block_key)
	if err != nil {
		return "", err
	}
	return bson.ObjectIdHex(str), nil
}

// Builds a user from his Id and information in http_header.
func BuildUser(a iface.Filter, ev iface.Event, user_id bson.ObjectId, http_header map[string][]string) (map[string]interface{}, error) {
	user, err := FindUser(a, user_id)
	if err != nil || user == nil {
		user = EmptyUser()
	}
	_, langs_are_set := user["languages"]
	if !langs_are_set {
		langs, has := http_header["Accept-Language"]
		if has {
			user["languages"] = ParseAcceptLanguage(langs[0])
		} else {
			user["languages"] = []string{"en"}
		}
	}
	ev.Trigger("user.build", user)
	return user, nil
}

func hashPass(pass string) string {
	h := sha1.New()
	io.WriteString(h, pass)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Returns true if the username is still available.
func NameAvailable(a iface.Filter, name string) (bool, error) {
	q := bson.M{"slug": slugify.S(name)}
	users, err := a.AddQuery(q).Find()
	if err != nil {
		return false, err
	}
	if len(users) > 0 {
		return false, nil
	}
	return true, nil
}

func RegisterUser(a iface.Filter, user map[string]interface{}) (bson.ObjectId, error) {
	user["password"] = hashPass(user["password"].(string))
	if _, has := user["level"]; !has {
		user["level"] = 100
	}
	user_id := bson.NewObjectId()
	user["_id"] = user_id
	err := a.Insert(user)
	if err != nil {
		return "", fmt.Errorf("Name is not unique.")
	}
	return user_id, nil
}

// Function intended to encrypt the user id before storing it as a cookie.
// encr flag controls
// block_key must be secret.
func encDecStr(block_key []byte, value string, encr bool) (string, error) {
	if block_key == nil || len(block_key) == 0 || len(block_key) < block_size {
		return "", fmt.Errorf("Can't encrypt/decrypt: block key is not proper.")
	}
	if len(value) == 0 {
		return "", fmt.Errorf("Nothing to encrypt/decrypt.")
	}
	block_key = block_key[:block_size]
	block, err := aes.NewCipher(block_key)
	if err != nil {
		return "", err
	}
	var bs []byte
	if encr {
		bs, err = encrypt(block, []byte(value))
	} else {
		bs, err = decrypt(block, []byte(value))
	}
	if err != nil {
		return "", err
	}
	if bs == nil {
		return "", fmt.Errorf("Somethign went wrong when encoding/decoding.")
	} // Just in case.
	return string(bs), nil
}

// Encrypts a value and encodes it with base64.
func encryptStr(block_key []byte, value string) (string, error) {
	str, err := encDecStr(block_key, value, true)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(str)), nil
}

// Decodes a value with base64 and then decrypts it.
func decryptStr(block_key []byte, value string) (string, error) {
	decoded_b, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return encDecStr(block_key, string(decoded_b), false)
}

// The following functions are taken from securecookie package of the Gorilla web toolkit made by Rodrigo Moraes.
// Only modification was to make the GenerateRandomKey function private.

// encrypt encrypts a value using the given block in counter mode.
//
// A random initialization vector (http://goo.gl/zF67k) with the length of the
// block size is prepended to the resulting ciphertext.
func encrypt(block cipher.Block, value []byte) ([]byte, error) {
	iv := generateRandomKey(block.BlockSize())
	if iv == nil {
		return nil, errors.New("securecookie: failed to generate random iv")
	}
	// Encrypt it.
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(value, value)
	// Return iv + ciphertext.
	return append(iv, value...), nil
}

// decrypt decrypts a value using the given block in counter mode.
//
// The value to be decrypted must be prepended by a initialization vector
// (http://goo.gl/zF67k) with the length of the block size.
func decrypt(block cipher.Block, value []byte) ([]byte, error) {
	size := block.BlockSize()
	if len(value) > size {
		// Extract iv.
		iv := value[:size]
		// Extract ciphertext.
		value = value[size:]
		// Decrypt it.
		stream := cipher.NewCTR(block, iv)
		stream.XORKeyStream(value, value)
		return value, nil
	}
	return nil, errors.New("securecookie: the value could not be decrypted")
}

// GenerateRandomKey creates a random key with the given strength.
func generateRandomKey(strength int) []byte {
	k := make([]byte, strength)
	if _, err := rand.Read(k); err != nil {
		return nil
	}
	return k
}
