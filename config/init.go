package config

import (
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Instance 配置实例
var Instance *Config

// Init 初始化配置
func Init(cfgFile string, workDir string) {
	viper.Reset()
	viper.SetConfigType("yaml")

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find config file in path.
		viper.AddConfigPath(".")
		viper.SetConfigName("artalk-go.yml")
	}

	viper.SetEnvPrefix("ATG")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 切换工作目录
	if workDir != "" {
		viper.AddConfigPath(workDir) // must before
		if err := os.Chdir(workDir); err != nil {
			logrus.Fatal("工作目录切换错误 ", err)
		}
	}

	if err := viper.ReadInConfig(); err == nil {
		// fmt.Print("\n")
		// fmt.Println("- Using ArtalkGo config file:", viper.ConfigFileUsed())
	} else {
		logrus.Fatal("找不到配置文件，使用 `-h` 参数查看帮助")
	}

	Instance = &Config{}
	err := viper.Unmarshal(&Instance)
	if err != nil {
		logrus.Errorf("unable to decode into struct, %v", err)
	}

	// 后续处理
	postInit()
}

func postInit() {
	// 检查 app_key 是否设置
	if strings.TrimSpace(Instance.AppKey) == "" {
		logrus.Fatal("请检查配置文件，并设置一个 app_key (任意字符串) 用于数据加密")
	}

	// 设置时区
	if strings.TrimSpace(Instance.TimeZone) == "" {
		logrus.Fatal("请检查配置文件，并设置 timezone")
	}
	denverLoc, _ := time.LoadLocation(Instance.TimeZone)
	time.Local = denverLoc

	// 默认站点配置
	Instance.SiteDefault = strings.TrimSpace(Instance.SiteDefault)
	if Instance.SiteDefault == "" {
		logrus.Fatal("请设置 SiteDefault 默认站点，不能为空")
	}

	// 缓存配置
	if Instance.Cache.Type == "" {
		// 默认使用内建缓存
		Instance.Cache.Type = CacheTypeBuiltin
	}
	if Instance.Cache.Type != CacheTypeDisabled {
		// 非缓存禁用模式，Enabled = true
		Instance.Cache.Enabled = true
	}

	// 配置文件 alias 处理
	if Instance.Captcha.ActionLimit == 0 {
		Instance.Captcha.Always = true
	}

	/* 检查废弃需更新配置 */
	if Instance.Captcha.ActionTimeout != 0 {
		logrus.Warn("captcha.action_timeout 配置项已废弃，请使用 captcha.action_reset 代替")
		if Instance.Captcha.ActionReset == 0 {
			Instance.Captcha.ActionReset = Instance.Captcha.ActionTimeout
		}
	}
	if len(Instance.AllowOrigins) != 0 {
		logrus.Warn("allow_origins 配置项已废弃，请使用 trusted_domains 代替")
		if len(Instance.TrustedDomains) == 0 {
			Instance.TrustedDomains = Instance.AllowOrigins
		}
	}

	// @version < 2.2.0
	if Instance.Notify != nil {
		logrus.Warn("notify 配置项已废弃，请使用 admin_notify 代替")
		Instance.AdminNotify = *Instance.Notify
	}
	if Instance.AdminNotify.Email == nil {
		Instance.AdminNotify.Email = &AdminEmailConf{
			Enabled: true, // 默认开启管理员邮件通知
		}
	}
	if Instance.Email.MailSubjectToAdmin != "" {
		logrus.Warn("email.mail_subject_to_admin 配置项已废弃，请使用 admin_notify.email.mail_subject 代替")
		Instance.AdminNotify.Email.MailSubject = Instance.Email.MailSubjectToAdmin
	}

	// 管理员邮件通知配置继承
	if Instance.AdminNotify.Email.MailSubject == "" {
		if Instance.AdminNotify.NotifySubject != "" {
			Instance.AdminNotify.Email.MailSubject = Instance.AdminNotify.NotifySubject
		} else if Instance.Email.MailSubject != "" {
			Instance.AdminNotify.Email.MailSubject = Instance.Email.MailSubject
		}
	}

	// 默认待审模式下开启管理员通知嘈杂模式，保证管理员能看到待审核文章
	if Instance.Moderator.PendingDefault {
		Instance.AdminNotify.NoiseMode = true
	}
}
