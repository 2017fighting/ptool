package config

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/gofrs/flock"
	"github.com/jpillora/go-tld"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/sagan/ptool/util"
)

const (
	BRUSH_CAT        = "_brush"
	XSEED_TAG        = "_xseed"
	NOADD_TAG        = "_noadd"
	NOXSEED_TAG      = "noxseed" // BT 客户端里含有此 tag 的种子不会被辅种
	STATS_FILENAME   = "ptool_stats.txt"
	HISTORY_FILENAME = "ptool_history"

	DEFAULT_TIMEOUT                                 = int64(5)
	DEFAULT_SHELL_MAX_SUGGESTIONS                   = int64(5)
	DEFAULT_SHELL_MAX_HISTORY                       = int64(500)
	DEFAULT_SITE_TIMEZONE                           = "Asia/Shanghai"
	DEFAULT_CLIENT_BRUSH_MIN_DISK_SPACE             = int64(5 * 1024 * 1024 * 1024)
	DEFAULT_CLIENT_BRUSH_SLOW_UPLOAD_SPEED_TIER     = int64(100 * 1024)
	DEFAULT_CLIENT_BRUSH_MAX_DOWNLOADING_TORRENTS   = int64(6)
	DEFAULT_CLIENT_BRUSH_MAX_TORRENTS               = int64(9999)
	DEFAULT_CLIENT_BRUSH_MIN_RATION                 = float64(0.2)
	DEFAULT_CLIENT_BRUSH_DEFAULT_UPLOAD_SPEED_LIMIT = int64(10 * 1024 * 1024)
	DEFAULT_SITE_TIMEOUT                            = DEFAULT_TIMEOUT
	DEFAULT_SITE_BRUSH_TORRENT_MIN_SIZE_LIMIT       = int64(0)
	DEFAULT_SITE_BRUSH_TORRENT_MAX_SIZE_LIMIT       = int64(1024 * 1024 * 1024 * 1024 * 1024) // 1PB, that's say, unlimited
	DEFAULT_SITE_TORRENT_UPLOAD_SPEED_LIMIT         = int64(10 * 1024 * 1024)
	DEFAULT_SITE_FLOW_CONTROL_INTERVAL              = int64(3)
	DEFAULT_SITE_MAX_REDIRECTS                      = int64(3)
	DEFAULT_COOKIECLOUD_TIMEOUT                     = DEFAULT_TIMEOUT
)

type CookiecloudConfigStruct struct {
	Name     string   `yaml:"name"`
	Disabled bool     `yaml:"disabled"`
	Server   string   `yaml:"server"` // CookieCloud API Server Url (with API_ROOT, if exists)
	Uuid     string   `yaml:"uuid"`
	Password string   `yaml:"password"`
	Proxy    string   `yaml:"proxy"`
	Sites    []string `yaml:"sites"`
	Timeoout int64    `yaml:"timeout"`
	Comment  string   `yaml:"comment"`
}

type GroupConfigStruct struct {
	Name    string   `yaml:"name"`
	Sites   []string `yaml:"sites"`
	Comment string   `yaml:"comment"`
}

type AliasConfigStruct struct {
	Name string `yaml:"name"`
	Cmd  string `yaml:"cmd"`
}

type ClientConfigStruct struct {
	Type                              string  `yaml:"type"`
	Name                              string  `yaml:"name"`
	Comment                           string  `yaml:"comment"`
	Disabled                          bool    `yaml:"disabled"`
	Url                               string  `yaml:"url"`
	Username                          string  `yaml:"username"`
	Password                          string  `yaml:"password"`
	BrushMinDiskSpace                 string  `yaml:"brushMinDiskSpace"`
	BrushSlowUploadSpeedTier          string  `yaml:"brushSlowUploadSpeedTier"`
	BrushMaxDownloadingTorrents       int64   `yaml:"brushMaxDownloadingTorrents"`
	BrushMaxTorrents                  int64   `yaml:"brushMaxTorrents"`
	BrushMinRatio                     float64 `yaml:"brushMinRatio"`
	BrushDefaultUploadSpeedLimit      string  `yaml:"brushDefaultUploadSpeedLimit"`
	BrushMinDiskSpaceValue            int64
	BrushSlowUploadSpeedTierValue     int64
	BrushDefaultUploadSpeedLimitValue int64
	QbittorrentNoLogin                bool `yaml:"qbittorrentNoLogin"`  // if set, will NOT send login request
	QbittorrentNoLogout               bool `yaml:"qbittorrentNoLogout"` // if set, will NOT send logout request
}

type SiteConfigStruct struct {
	Type                           string     `yaml:"type"`
	Name                           string     `yaml:"name"`
	Aliases                        []string   // for internal use only
	Comment                        string     `yaml:"comment"`
	Disabled                       bool       `yaml:"disabled"`
	Hidden                         bool       `yaml:"hidden"` // exclude from default groups (like "_all")
	Url                            string     `yaml:"url"`
	Domains                        []string   `yaml:"domains"` // other site domains (do not include subdomain part)
	TorrentsUrl                    string     `yaml:"torrentsUrl"`
	SearchUrl                      string     `yaml:"searchUrl"`
	SearchQueryVariable            string     `yaml:"searchQueryVariable"`
	TorrentsExtraUrls              []string   `yaml:"torrentsExtraUrls"`
	Cookie                         string     `yaml:"cookie"`
	UserAgent                      string     `yaml:"userAgent"`
	Impersonate                    string     `yaml:"impersonate"`
	HttpHeaders                    [][]string `yaml:"httpHeaders"`
	Ja3                            string     `yaml:"ja3"`
	Timeoout                       int64      `yaml:"timeout"`
	H2Fingerprint                  string     `yaml:"h2Fingerprint"`
	Proxy                          string     `yaml:"proxy"`
	Insecure                       bool       `yaml:"insecure"` // 访问站点时强制跳过TLS证书安全校验
	Secure                         bool       `yaml:"secure"`   // 访问站点时强制TLS证书安全校验
	TorrentUploadSpeedLimit        string     `yaml:"torrentUploadSpeedLimit"`
	GlobalHnR                      bool       `yaml:"globalHnR"`
	Timezone                       string     `yaml:"timezone"`
	BrushTorrentMinSizeLimit       string     `yaml:"brushTorrentMinSizeLimit"`
	BrushTorrentMaxSizeLimit       string     `yaml:"brushTorrentMaxSizeLimit"`
	BrushAllowNoneFree             bool       `yaml:"brushAllowNoneFree"`
	BrushAllowPaid                 bool       `yaml:"brushAllowPaid"`
	BrushAllowHr                   bool       `yaml:"brushAllowHr"`
	BrushAllowZeroSeeders          bool       `yaml:"brushAllowZeroSeeders"`
	BrushExcludes                  []string   `yaml:"brushExcludes"`
	SelectorTorrentsListHeader     string     `yaml:"selectorTorrentsListHeader"`
	SelectorTorrentsList           string     `yaml:"selectorTorrentsList"`
	SelectorTorrentBlock           string     `yaml:"selectorTorrentBlock"` // dom block of a torrent in list
	SelectorTorrent                string     `yaml:"selectorTorrent"`
	SelectorTorrentDownloadLink    string     `yaml:"selectorTorrentDownloadLink"`
	SelectorTorrentDetailsLink     string     `yaml:"selectorTorrentDetailsLink"`
	SelectorTorrentTime            string     `yaml:"selectorTorrentTime"`
	SelectorTorrentSeeders         string     `yaml:"selectorTorrentSeeders"`
	SelectorTorrentLeechers        string     `yaml:"selectorTorrentLeechers"`
	SelectorTorrentSnatched        string     `yaml:"selectorTorrentSnatched"`
	SelectorTorrentSize            string     `yaml:"selectorTorrentSize"`
	SelectorTorrentProcessBar      string     `yaml:"selectorTorrentProcessBar"`
	SelectorTorrentFree            string     `yaml:"SelectorTorrentFree"`
	SelectorTorrentNoTraffic       string     `yaml:"selectorTorrentNoTraffic"`
	SelectorTorrentNeutral         string     `yaml:"selectorTorrentNeutral"`
	SelectorTorrentHnR             string     `yaml:"selectorTorrentHnR"`
	SelectorTorrentPaid            string     `yaml:"selectorTorrentPaid"`
	SelectorTorrentDiscountEndTime string     `yaml:"selectorTorrentDiscountEndTime"`
	SelectorUserInfo               string     `yaml:"selectorUserInfo"`
	SelectorUserInfoUserName       string     `yaml:"selectorUserInfoUserName"`
	SelectorUserInfoUploaded       string     `yaml:"selectorUserInfoUploaded"`
	SelectorUserInfoDownloaded     string     `yaml:"selectorUserInfoDownloaded"`
	TorrentDownloadUrl             string     `yaml:"torrentDownloadUrl"` // use {id} placeholders in url
	TorrentDownloadUrlPrefix       string     `yaml:"torrentDownloadUrlPrefix"`
	Passkey                        string     `yaml:"passkey"`
	UseCuhash                      bool       `yaml:"useCuhash"`    // hdcity 使用机制。种子下载地址里必须有cuhash参数
	UseDigitHash                   bool       `yaml:"useDigitHash"` // ttg 使用机制。种子下载地址末段必须有4位数字校验码或Passkey参数(即使有 Cookie)
	TorrentUrlIdRegexp             string     `yaml:"torrentUrlIdRegexp"`
	FlowControlInterval            int64      `yaml:"flowControlInterval"` // 暂定名。两次请求种子列表页间隔时间(秒)
	NexusphpNoLetDown              bool       `yaml:"nexusphpNoLetDown"`
	MaxRedirects                   int64      `yaml:"maxRedirects"`
	TorrentUploadSpeedLimitValue   int64
	BrushTorrentMinSizeLimitValue  int64
	BrushTorrentMaxSizeLimitValue  int64
	AutoComment                    string // 自动更新 ptool.toml 时系统生成的 comment。会被写入 Comment 字段
}

type ConfigStruct struct {
	Hushshell           bool                       `yaml:"hushshell"`
	ShellMaxSuggestions int64                      `yaml:"shellMaxSuggestions"`
	ShellMaxHistory     int64                      `yaml:"shellMaxHistory"`
	IyuuToken           string                     `yaml:"iyuuToken"`
	SiteProxy           string                     `yaml:"siteProxy"`
	SiteUserAgent       string                     `yaml:"siteUserAgent"`
	SiteImpersonate     string                     `yaml:"siteImpersonate"`
	SiteHttpHeaders     [][]string                 `yaml:"siteHttpHeaders"`
	SiteJa3             string                     `yaml:"siteJa3"`
	SiteTimeout         int64                      `yaml:"siteTimeout"` // 访问网站超时时间(秒)
	SiteSecure          bool                       `yaml:"siteSecure"`  // 强制开启所有站点 TLS 证书校验。
	SiteH2Fingerprint   string                     `yaml:"siteH2Fingerprint"`
	BrushEnableStats    bool                       `yaml:"brushEnableStats"`
	Clients             []*ClientConfigStruct      `yaml:"clients"`
	Sites               []*SiteConfigStruct        `yaml:"sites"`
	Groups              []*GroupConfigStruct       `yaml:"groups"`
	Aliases             []*AliasConfigStruct       `yaml:"aliases"`
	Cookieclouds        []*CookiecloudConfigStruct `yaml:"cookieclouds"`
	Comment             string                     `yaml:"comment"`
	ClientsEnabled      []*ClientConfigStruct
	SitesEnabled        []*SiteConfigStruct
}

var (
	VerboseLevel                        = 0
	InShell                             = false
	Initialized                         = false
	ConfigDir                           = "" // "/root/.config/ptool"
	ConfigFile                          = "" // "ptool.toml"
	DefaultConfigFile                   = "" // set when start
	ConfigName                          = "" // "ptool"
	ConfigType                          = "" // "toml"
	LockFile                            = ""
	LockOrExit                          = false
	Fork                                = false
	configData            *ConfigStruct = &ConfigStruct{}
	clientsConfigMap                    = map[string]*ClientConfigStruct{}
	sitesConfigMap                      = map[string]*SiteConfigStruct{}
	aliasesConfigMap                    = map[string]*AliasConfigStruct{}
	groupsConfigMap                     = map[string]*GroupConfigStruct{}
	cookiecloudsConfigMap               = map[string]*CookiecloudConfigStruct{}
	once                  sync.Once
)

func init() {
}

// Update configed sites in place, merge the provided (updated) sites with existing config.
func UpdateSites(updatesites []*SiteConfigStruct) {
	if len(updatesites) == 0 {
		return
	}
	allsites := Get().Sites
	for _, updatesite := range updatesites {
		if updatesite.AutoComment != "" {
			m := regexp.MustCompile(`^(.*?)<!--\{ptool\}.*?-->(.*)$`)
			autoComment := fmt.Sprintf(`<!--{ptool} %s-->`, updatesite.AutoComment)
			comment := m.ReplaceAllString(updatesite.Comment, fmt.Sprintf(`$1%s$2`, autoComment))
			if comment == updatesite.Comment {
				updatesite.Comment += autoComment
			} else {
				updatesite.Comment = comment
			}
		}

		updatesite.Register()
		index := slices.IndexFunc(allsites, func(scs *SiteConfigStruct) bool {
			return scs.GetName() == updatesite.GetName()
		})
		if index != -1 {
			util.Assign(allsites[index], updatesite, nil)
		} else {
			allsites = append(allsites, updatesite)
		}
	}
	configData.Sites = allsites
	configData.UpdateSitesDerivative()
}

// Re-write the whole config file using memory data.
// Currently, only sites will be overrided.
// Due to technical limitations, all existing comments will be LOST.
// For now, new config data will NOT take effect for current ptool process.
func Set() error {
	lock := flock.New(path.Join(ConfigDir, "ptool.lock"))
	if ok, err := lock.TryLock(); err != nil || !ok {
		return fmt.Errorf("unable to lock config file: err=%v", err)
	}
	defer lock.Unlock()
	sites := Get().Sites
	newsites := []map[string]any{}
	for i := range sites {
		newsite := util.StructToMap(*sites[i], true, true)
		newsites = append(newsites, newsite)
	}
	viper.Set("sites", newsites)
	return viper.WriteConfig()
}

func Get() *ConfigStruct {
	once.Do(func() {
		log.Debugf("Read config file %s/%s", ConfigDir, ConfigFile)
		viper.SetConfigName(ConfigName)
		viper.SetConfigType(ConfigType)
		viper.AddConfigPath(ConfigDir)
		viper.SetDefault("ShellMaxSuggestions", DEFAULT_SHELL_MAX_SUGGESTIONS)
		viper.SetDefault("ShellMaxHistory", DEFAULT_SHELL_MAX_HISTORY)
		err := viper.ReadInConfig()
		if err != nil {
			log.Fatalf("Fail to read config file: %v", err)
		}
		err = viper.Unmarshal(&configData)
		if err != nil {
			log.Fatalf("Fail to parse config file: %v", err)
		}

		for _, client := range configData.Clients {
			v, err := util.RAMInBytes(client.BrushMinDiskSpace)
			if err != nil || v < 0 {
				v = DEFAULT_CLIENT_BRUSH_MIN_DISK_SPACE
			}
			client.BrushMinDiskSpaceValue = v

			v, err = util.RAMInBytes(client.BrushSlowUploadSpeedTier)
			if err != nil || v <= 0 {
				v = DEFAULT_CLIENT_BRUSH_SLOW_UPLOAD_SPEED_TIER
			}
			client.BrushSlowUploadSpeedTierValue = v

			v, err = util.RAMInBytes(client.BrushDefaultUploadSpeedLimit)
			if err != nil || v <= 0 {
				v = DEFAULT_CLIENT_BRUSH_DEFAULT_UPLOAD_SPEED_LIMIT
			}
			client.BrushDefaultUploadSpeedLimitValue = v

			if client.Url != "" {
				urlObj, err := url.Parse(client.Url)
				if err != nil {
					log.Fatalf("Failed to parse client %s url config: %v", client.Name, err)
				}
				client.Url = urlObj.String()
			}

			if client.BrushMaxDownloadingTorrents == 0 {
				client.BrushMaxDownloadingTorrents = DEFAULT_CLIENT_BRUSH_MAX_DOWNLOADING_TORRENTS
			}

			if client.BrushMaxTorrents == 0 {
				client.BrushMaxTorrents = DEFAULT_CLIENT_BRUSH_MAX_TORRENTS
			}

			if client.BrushMinRatio == 0 {
				client.BrushMinRatio = DEFAULT_CLIENT_BRUSH_MIN_RATION
			}

			if client.Name == "" {
				log.Fatalf("Invalid config file: client name can not be empty")
			}
			if clientsConfigMap[client.Name] != nil {
				log.Fatalf("Invalid config file: duplicate client name %s found", client.Name)
			}
			clientsConfigMap[client.Name] = client
		}
		for _, site := range configData.Sites {
			if sitesConfigMap[site.GetName()] != nil {
				log.Fatalf("Invalid config file: duplicate site name %s found", site.GetName())
			}
			site.Register()
		}
		for _, group := range configData.Groups {
			if group.Name == "" {
				log.Fatalf("Invalid config file: group name can not be empty for %v", group)
			}
			if groupsConfigMap[group.Name] != nil {
				log.Fatalf("Invalid config file: duplicate group name %s found", group.Name)
			}
			groupsConfigMap[group.Name] = group
		}
		for _, alias := range configData.Aliases {
			if alias.Name == "" {
				log.Fatalf("Invalid config file: alias name can not be empty for %v", alias)
			}
			if alias.Name == "alias" {
				log.Fatalf("Invalid config file: alias name can not be 'alias' itself")
			}
			if aliasesConfigMap[alias.Name] != nil {
				log.Fatalf("Invalid config file: duplicate alias name %s found", alias.Name)
			}
			aliasesConfigMap[alias.Name] = alias
		}
		for _, cookiecloud := range configData.Cookieclouds {
			if cookiecloud.Name == "" {
				continue
			}
			if cookiecloudsConfigMap[cookiecloud.Name] != nil {
				log.Fatalf("Invalid config file: duplicate cookiecloud name %s found", cookiecloud.Name)
			}
			cookiecloudsConfigMap[cookiecloud.Name] = cookiecloud
		}
		configData.ClientsEnabled = util.Filter(configData.Clients, func(c *ClientConfigStruct) bool {
			return !c.Disabled
		})
		configData.UpdateSitesDerivative()
	})
	return configData
}

func GetClientConfig(name string) *ClientConfigStruct {
	Get()
	if name == "" {
		return nil
	}
	return clientsConfigMap[name]
}

func GetSiteConfig(name string) *SiteConfigStruct {
	Get()
	if name == "" {
		return nil
	}
	return sitesConfigMap[name]
}

func GetGroupConfig(name string) *GroupConfigStruct {
	Get()
	if name == "" {
		return nil
	}
	return groupsConfigMap[name]
}

func GetAliasConfig(name string) *AliasConfigStruct {
	Get()
	if name == "" {
		return nil
	}
	return aliasesConfigMap[name]
}

func GetCookiecloudConfig(name string) *CookiecloudConfigStruct {
	Get()
	if name == "" {
		return nil
	}
	return cookiecloudsConfigMap[name]
}

// if name is a group, return it's sites, otherwise return nil
func GetGroupSites(name string) []string {
	if name == "_all" { // special group of all sites
		sitenames := []string{}
		for _, siteConfig := range Get().SitesEnabled {
			if siteConfig.Hidden {
				continue
			}
			sitenames = append(sitenames, siteConfig.GetName())
		}
		return sitenames
	}
	group := GetGroupConfig(name)
	if group != nil {
		return group.Sites
	}
	return nil
}

func ParseGroupAndOtherNamesWithoutDeduplicate(names ...string) []string {
	names2 := []string{}
	for _, name := range names {
		groupSites := GetGroupSites(name)
		if groupSites != nil {
			names2 = append(names2, groupSites...)
		} else {
			names2 = append(names2, name)
		}
	}
	return names2
}

// parse an slice of groupOrOther names, expand group name to site names, return the final slice of names
func ParseGroupAndOtherNames(names ...string) []string {
	names = ParseGroupAndOtherNamesWithoutDeduplicate(names...)
	return util.UniqueSlice(names)
}

// Generate derivative info from site config and register itself
func (siteConfig *SiteConfigStruct) Register() {
	v, err := util.RAMInBytes(siteConfig.TorrentUploadSpeedLimit)
	if err != nil || v <= 0 {
		v = DEFAULT_SITE_TORRENT_UPLOAD_SPEED_LIMIT
	}
	siteConfig.TorrentUploadSpeedLimitValue = v

	if siteConfig.Url != "" {
		urlObj, err := url.Parse(siteConfig.Url)
		if err != nil {
			log.Fatalf("Failed to parse site %s url config: %v", siteConfig.GetName(), err)
		}
		siteConfig.Url = urlObj.String()
	}

	v, err = util.RAMInBytes(siteConfig.BrushTorrentMinSizeLimit)
	if err != nil || v <= 0 {
		v = DEFAULT_SITE_BRUSH_TORRENT_MIN_SIZE_LIMIT
	}
	siteConfig.BrushTorrentMinSizeLimitValue = v

	v, err = util.RAMInBytes(siteConfig.BrushTorrentMaxSizeLimit)
	if err != nil || v <= 0 {
		v = DEFAULT_SITE_BRUSH_TORRENT_MAX_SIZE_LIMIT
	}
	siteConfig.BrushTorrentMaxSizeLimitValue = v

	sitesConfigMap[siteConfig.GetName()] = siteConfig
}

func (siteConfig *SiteConfigStruct) GetName() string {
	id := siteConfig.Name
	if id == "" {
		id = siteConfig.Type
	}
	return id
}

func (siteConfig *SiteConfigStruct) GetTimezone() string {
	tz := siteConfig.Timezone
	if tz == "" {
		tz = DEFAULT_SITE_TIMEZONE
	}
	return tz
}

// parse a site internal url (e.g.: special.php), return absolute url
func (siteConfig *SiteConfigStruct) ParseSiteUrl(siteUrl string, appendQueryStringDelimiter bool) string {
	pageUrl := ""
	if siteUrl != "" {
		if util.IsUrl(siteUrl) {
			pageUrl = siteUrl
		} else {
			siteUrl = strings.TrimPrefix(siteUrl, "/")
			pageUrl = strings.TrimSuffix(siteConfig.Url, "/") + "/" + siteUrl
		}
	}

	if appendQueryStringDelimiter {
		pageUrl = util.AppendUrlQueryStringDelimiter(pageUrl)
	}
	return pageUrl
}

func MatchSite(domain string, siteConfig *SiteConfigStruct) bool {
	if domain == "" {
		return false
	}
	if siteConfig.Url != "" {
		u, err := tld.Parse(siteConfig.Url)
		if err != nil {
			return false
		}
		siteDomain := u.Domain + "." + u.TLD
		if domain == siteDomain {
			return true
		}
	}
	for _, siteDomain := range siteConfig.Domains {
		if siteDomain == domain {
			return true
		}
	}
	return false
}

func (configData *ConfigStruct) UpdateSitesDerivative() {
	configData.SitesEnabled = util.Filter(configData.Sites, func(s *SiteConfigStruct) bool {
		return !s.Disabled
	})
}
