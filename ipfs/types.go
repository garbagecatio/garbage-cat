package ipfs

type Arc3 struct {
	Name           string                  `json:"name"`
	Decimals       int64                   `json:"decimals"`
	Description    string                  `json:"description"`
	Image          string                  `json:"image"`
	ImageIntegrity string                  `json:"image_integrity"`
	ImageMimetype  string                  `json:"image_mimetype"`
	UnitName       string                  `json:"unitName"`
	AssetName      string                  `json:"assetName"`
	Properties     *map[string]interface{} `json:"properties"`
}

type Arc69 struct {
	Standard   string        `json:"standard"`
	Attributes []interface{} `json:"attributes"`
}

type Collection struct {
	Name string `json:"name"`
}

type Creator struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Address     string `json:"address"`
}

type Royalty struct {
	Name  string `json:"name"`
	Addr  string `json:"addr"`
	Share int64  `json:"share"`
}

const ARC19_MINTING_TEMPLATE = "template-ipfs://{ipfscid:1:raw:reserve:sha2-256}"

type Arc19 struct {
	Name           string                  `json:"name"`
	Description    *string                 `json:"description"`
	Standard       string                  `json:"standard"`
	Decimals       *int64                  `json:"decimals"`
	Image          string                  `json:"image"`
	ImageMimetype  string                  `json:"image_mimetype"`
	ImageIntegrity *string                 `json:"image_integrity"`
	Properties     *map[string]interface{} `json:"properties"`
}
