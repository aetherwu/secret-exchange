package wechat

import (
	"image"
	"strings"

	"github.com/medivhzhan/weapp/code"
)

// GetQRCode 生成小程序码
func GetQRCode(page, scene string) (image.Image, error) {
	coder := code.QRCoder{
		Scene:     scene,
		Page:      page,  // 识别二维码后进入小程序的页面链接
		Width:     430,   // 图片宽度
		IsHyaline: false, // 是否需要透明底色
		AutoColor: false, // 自动配置线条颜色, 如果颜色依然是黑色, 则说明不建议配置主色调
		LineColor: code.Color{ //  AutoColor 为 false 时生效, 使用 rgb 设置颜色 十进制表示
			R: "0",
			G: "0",
			B: "0",
		},
	}

	token, err := GetWeChatAccessToken()
	if err != nil {
		return nil, err
	}
	res, err := coder.UnlimitedAppCode(token)
	if err != nil {
		if strings.Contains(err.Error(), "token") {
			token, err = ForceGetWeChatAccessToken()
			if err != nil {
				return nil, err
			}
			res, err = coder.UnlimitedAppCode(token)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	defer res.Body.Close()

	img, _, err := image.Decode(res.Body)
	return img, err
}
