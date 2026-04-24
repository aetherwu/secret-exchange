package draw

import (
	"dagong.in/delphi-web-server/models"
)

// All draw functions are stubbed — WeChat share images are not used in the web version.

func GenWeChatShareImageOfDraw(qa models.QADisplay) ([]byte, error) {
	return nil, nil
}

func GenWeChatShareImageOfUserDraw(user models.User) ([]byte, error) {
	return nil, nil
}

func GenProfileWeChatShareImage(avatar string, selfIntro string, nickname string) ([]byte, error) {
	return nil, nil
}

func GenProfileQRCodeWeChatShareImage(nickname string, openID string, avatar string, totalQuestionCount int) ([]byte, error) {
	return nil, nil
}

func GenAnswerQRCodeWeChatShareImage(qa models.QADisplay) ([]byte, error) {
	return nil, nil
}
