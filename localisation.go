package servermanager

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func ExampleLocaliser() {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yml", yaml.Unmarshal)

	enMessageFile, err := bundle.LoadMessageFile("./localisation/en.yml")

	if err != nil {
		logrus.WithError(err).Error("couldn't load message file")
	}

	err = bundle.AddMessages(language.English, enMessageFile.Messages...)

	if err != nil {
		logrus.WithError(err).Error("couldn't add messages")
	}

	esMessageFile, err := bundle.LoadMessageFile("./localisation/es.yml")

	if err != nil {
		logrus.WithError(err).Error("couldn't load message file")
	}

	err = bundle.AddMessages(language.Spanish, esMessageFile.Messages...)

	if err != nil {
		logrus.WithError(err).Error("couldn't add messages")
	}

	{
		localiser := i18n.NewLocalizer(bundle, "en")
		fmt.Println(localiser.MustLocalize(&i18n.LocalizeConfig{MessageID: "HelloWorld"}))
	}
	{
		localiser := i18n.NewLocalizer(bundle, "es")
		fmt.Println(localiser.MustLocalize(&i18n.LocalizeConfig{MessageID: "HelloWorld"}))
	}
}