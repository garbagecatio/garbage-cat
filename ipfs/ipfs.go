package ipfs

import (
	"crypto/sha256"
	"encoding/base64"
	baseErrors "errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/garbagecatio/rug-ninja-sniper/errors"

	"github.com/algorand/go-algorand-sdk/v2/types"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
)

var (
	ErrUnknownSpec      = baseErrors.New("unsupported template-ipfs spec")
	ErrUnsupportedField = baseErrors.New("unsupported ipfscid field, only reserve is currently supported")
	ErrUnsupportedCodec = baseErrors.New("unknown multicodec type in ipfscid spec")
	ErrUnsupportedHash  = baseErrors.New("unknown hash type in ipfscid spec")
	ErrInvalidV0        = baseErrors.New("cid v0 must always be dag-pb and sha2-256 codec/hash type")
	ErrHashEncoding     = baseErrors.New("error encoding new hash")
	templateIPFSRegexp  = regexp.MustCompile(`template-ipfs://{ipfscid:(?P<version>[01]):(?P<codec>[a-z0-9\-]+):(?P<field>[a-z0-9\-]+):(?P<hash>[a-z0-9\-]+)}`)
)

func ReserveAddressFromCID(cidToEncode cid.Cid) (string, error) {
	const op errors.Op = "ReserveAddressFromCID"
	decodedMultiHash, err := multihash.Decode(cidToEncode.Hash())
	if err != nil {
		return "", errors.E(op, fmt.Errorf("failed to decode ipfs cid: %w", err))
	}
	return types.EncodeAddress(decodedMultiHash.Digest)
}

func ParseASAUrl(asaUrl string, reserveAddress types.Address) (string, error) {
	const op errors.Op = "ParseASAUrl"

	matches := templateIPFSRegexp.FindStringSubmatch(asaUrl)
	if matches == nil {
		if strings.HasPrefix(asaUrl, "template-ipfs://") {
			return "", errors.E(op, ErrUnknownSpec)
		}
		return asaUrl, nil
	}
	if matches[templateIPFSRegexp.SubexpIndex("field")] != "reserve" {
		return "", errors.E(op, ErrUnsupportedField)
	}
	var (
		codec         multicodec.Code
		multihashType uint64
		hash          []byte
		err           error
		cidResult     cid.Cid
	)
	if err = codec.Set(matches[templateIPFSRegexp.SubexpIndex("codec")]); err != nil {
		return "", errors.E(op, ErrUnsupportedCodec)
	}
	multihashType = multihash.Names[matches[templateIPFSRegexp.SubexpIndex("hash")]]
	if multihashType == 0 {
		return "", errors.E(op, ErrUnsupportedHash)
	}

	hash, err = multihash.Encode(reserveAddress[:], multihashType)
	if err != nil {
		return "", errors.E(op, ErrHashEncoding)
	}
	if matches[templateIPFSRegexp.SubexpIndex("version")] == "0" {
		if codec != multicodec.DagPb {
			return "", errors.E(op, ErrInvalidV0)
		}
		if multihashType != multihash.SHA2_256 {
			return "", errors.E(op, ErrInvalidV0)
		}
		cidResult = cid.NewCidV0(hash)
	} else {
		cidResult = cid.NewCidV1(uint64(codec), hash)
	}
	return fmt.Sprintf("ipfs://%s", strings.ReplaceAll(asaUrl, matches[0], cidResult.String())), nil
}

func GetCID(url string) (string, error) {
	const op errors.Op = "getIPFSCID"

	url = strings.Replace(url, "?preview=1", "", -1)
	url = strings.Replace(url, "/?preview=1", "", -1)

	if strings.Contains(url, "?") {
		url = strings.Split(url, "?")[0]
	}

	if strings.Contains(url, "ipfs://") {
		return strings.TrimSpace(strings.Replace(url, "ipfs://", "", 1)), nil
	} else if strings.Contains(url, "/ipfs/") {
		return strings.TrimSpace(strings.Split(url, "/ipfs/")[1]), nil
	} else if strings.Contains(url, ".ipfs.") {
		return strings.TrimSpace(strings.Replace(strings.Replace(strings.Split(url, ".ipfs.")[0], "https://", "", 1), "http://", "", 1)), nil
	} else {
		splitURL := strings.FieldsFunc(url, func(r rune) bool { return r == '/' || r == '.' })
		for _, slice := range splitURL {
			if len(slice) >= 46 {
				return strings.TrimSpace(slice), nil
			}
		}
		return "", errors.E(op, fmt.Errorf("failed to get CID from url"))
	}
}

func GetIPFSData(url string) ([]byte, error) {
	const op errors.Op = "GetIPFSData"
	if !strings.HasPrefix(url, "ipfs://") {
		fmt.Printf("invalid ipfs url: %s\n", url)
		// return nil, errors.E(op, fmt.Errorf("invalid ipfs url: %s", url))
	}

	url = strings.Replace(url, "ipfs://", "https://ipfs.algonode.xyz/ipfs/", 1)
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.E(op, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.E(op, err)
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		return nil, errors.E(op, fmt.Errorf("ipfs request failed: %s", resp.Status))
	}

	return body, nil
}

func MediaIntegrity(file io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	hashed := hash.Sum(nil)
	hashBase64 := base64.StdEncoding.EncodeToString(hashed)

	return "sha256-" + hashBase64, nil
}
