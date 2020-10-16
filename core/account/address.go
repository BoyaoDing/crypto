package account

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xuperchain/crypto/common/utils"
	"github.com/xuperchain/crypto/core/base58"
	"github.com/xuperchain/crypto/core/config"
	"github.com/xuperchain/crypto/core/hash"
)

var (
	TooSmallNumOfkeysError         = errors.New("The total num of keys should be greater than one")
	CurveParamNilError             = errors.New("curve input param is nil")
	NotExactTheSameCurveInputError = errors.New("the curve is not same as curve of members")
	KeyParamNotMatchError          = errors.New("key param not match")
)

// generate address for multi-signature / ring-signature algorithm
//为多个公钥生成地址，比如环签名地址，多重签名地址
func GetAddressFromPublicKeys(keys []*ecdsa.PublicKey) (string, error) {
	// 所有参与者需要使用同一条椭圆曲线
	curveCheckResult := checkCurveForPublicKeys(keys)
	if curveCheckResult == false {
		return "", NotExactTheSameCurveInputError
	}

	// 再计算需要被hash的data
	publicKeyMap := make(map[string]string)
	for _, key := range keys {
		publicKeyMap[key.X.String()] = key.Y.String()
	}

	data, err := json.Marshal(publicKeyMap)
	if err != nil {
		return "", err
	}

	address, err := getAddressFromKeyData(keys[0], data)

	return address, nil
}

//验证钱包地址是否和指定的公钥数组match
//如果成功，返回true和对应的密码学标记位；如果失败，返回false和默认的密码学标记位0
func VerifyAddressUsingPublicKeys(address string, pubs []*ecdsa.PublicKey) (bool, uint8) {
	//base58反解回byte[]数组
	slice := base58.Decode(address)

	//检查是否是合法的base58编码
	if len(slice) < 1 {
		return false, 0
	}
	//拿到密码学标记位
	byteVersion := slice[:1]
	nVersion := uint8(byteVersion[0])

	realAddress, error := GetAddressFromPublicKeys(pubs)
	if error != nil {
		return false, 0
	}

	if realAddress == address {
		return true, nVersion
	}

	return false, 0
}

// check whether all the public keys are using the same curve
// 检查是否所有的环签名验证参与者使用的都是同一条椭圆曲线
func checkCurveForPublicKeys(keys []*ecdsa.PublicKey) bool {
	curve := keys[0].Curve

	for _, key := range keys {
		if curve != key.Curve {
			return false
		}
	}

	return true
}

func getAddressFromKeyData(pub *ecdsa.PublicKey, data []byte) (string, error) {
	outputSha256 := hash.HashUsingSha256(data)
	OutputRipemd160 := hash.HashUsingRipemd160(outputSha256)

	//暂时只支持一个字节长度，也就是uint8的密码学标志位
	// 判断是否是nist标准的私钥
	nVersion := config.Nist

	switch pub.Params().Name {
	case config.CurveNist: // NIST
	case config.CurveGm: // 国密
		nVersion = config.Gm
	default: // 不支持的密码学类型
		return "", fmt.Errorf("This cryptography[%v] has not been supported yet.", pub.Params().Name)
	}

	bufVersion := []byte{byte(nVersion)}

	strSlice := make([]byte, len(bufVersion)+len(OutputRipemd160))
	copy(strSlice, bufVersion)
	copy(strSlice[len(bufVersion):], OutputRipemd160)

	//using double SHA256 for future risks
	checkCode := hash.DoubleSha256(strSlice)
	simpleCheckCode := checkCode[:4]

	slice := make([]byte, len(strSlice)+len(simpleCheckCode))
	copy(slice, strSlice)
	copy(slice[len(strSlice):], simpleCheckCode)

	//使用base58编码，手写不容易出错。
	//相比Base64，Base58不使用数字"0"，字母大写"O"，字母大写"I"，和字母小写"l"，以及"+"和"/"符号。
	strEnc := base58.Encode(slice)

	return strEnc, nil
}

//返回33位长度的地址
func GetAddressFromPublicKey(pub *ecdsa.PublicKey) (string, error) {
	//using SHA256 and Ripemd160 for hash summary
	data := elliptic.Marshal(pub.Curve, pub.X, pub.Y)

	address, err := getAddressFromKeyData(pub, data)

	return address, err
}

//验证钱包地址是否和指定的公钥match
//如果成功，返回true和对应的密码学标记位；如果失败，返回false和默认的密码学标记位0
func VerifyAddressUsingPublicKey(address string, pub *ecdsa.PublicKey) (bool, uint8) {
	//base58反解回byte[]数组
	slice := base58.Decode(address)

	//检查是否是合法的base58编码
	if len(slice) < 1 {
		return false, 0
	}
	//拿到密码学标记位
	byteVersion := slice[:1]
	nVersion := uint8(byteVersion[0])

	realAddress, error := GetAddressFromPublicKey(pub)
	if error != nil {
		return false, 0
	}

	if realAddress == address {
		return true, nVersion
	}

	return false, 0
}

//验证钱包地址是否是合法的格式
//如果成功，返回true和对应的密码学标记位；如果失败，返回false和默认的密码学标记位0
func CheckAddressFormat(address string) (bool, uint8) {
	//base58反解回byte[]数组
	slice := base58.Decode(address)

	//检查是否是合法的base58编码
	if len(slice) < 1 {
		return false, 0
	}
	//拿到简单校验码
	simpleCheckCode := slice[len(slice)-4:]

	checkContent := slice[:len(slice)-4]
	checkCode := hash.DoubleSha256(checkContent)
	realSimpleCheckCode := checkCode[:4]

	byteVersion := slice[:1]
	nVersion := uint8(byteVersion[0])

	if utils.BytesCompare(realSimpleCheckCode, simpleCheckCode) {
		return true, nVersion
	}

	return false, 0
}
