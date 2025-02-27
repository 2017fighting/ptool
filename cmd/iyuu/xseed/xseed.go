package xseed

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sagan/ptool/client"
	"github.com/sagan/ptool/cmd"
	"github.com/sagan/ptool/cmd/common"
	"github.com/sagan/ptool/cmd/iyuu"
	"github.com/sagan/ptool/config"
	"github.com/sagan/ptool/constants"
	"github.com/sagan/ptool/site"
	"github.com/sagan/ptool/util"
	"github.com/sagan/ptool/util/torrentutil"
)

var command = &cobra.Command{
	Use:         "xseed {client}...",
	Annotations: map[string]string{"cobra-prompt-dynamic-suggestions": "iyuu.xseed"},
	Short:       "Cross seed using iyuu API.",
	Long: `Cross seed using iyuu API.
By default it will add xseed torrents from All sites unless --include-sites or --exclude-sites flag is set.`,
	Args: cobra.MatchAll(cobra.MinimumNArgs(1), cobra.OnlyValidArgs),
	RunE: xseed,
}

var (
	dryRun             = false
	addPaused          = false
	check              = false
	slowMode           = false
	maxXseedTorrents   = int64(0)
	maxConsecutiveFail = int64(0)
	includeSites       = ""
	excludeSites       = ""
	category           = ""
	addCategory        = ""
	addTags            = ""
	tag                = ""
	filter             = ""
	minTorrentSizeStr  = ""
	maxTorrentSizeStr  = ""
	iyuuRequestServer  = ""
)

func init() {
	command.Flags().BoolVarP(&slowMode, "slow", "", false, "Slow mode. wait after handling each xseed torrent")
	command.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Dry run. Do NOT actually add xseed torrents to client")
	command.Flags().BoolVarP(&addPaused, "add-paused", "", false, "Add xseed torrents to client in paused state")
	command.Flags().BoolVarP(&check, "check", "", false, "Let client do hash checking when adding xseed torrents")
	command.Flags().Int64VarP(&maxXseedTorrents, "max-torrents", "", -1,
		"Number limit of xseed torrents added. -1 == no limit")
	command.Flags().Int64VarP(&maxConsecutiveFail, "max-consecutive-fail", "", 3,
		"After consecutive fails to download torrent from a site of this times, will skip that site afterwards. "+
			"Note a 404 error does NOT count as a fail. -1 = no limit (never skip)")
	command.Flags().StringVarP(&includeSites, "include-sites", "", "",
		"Only add xseed torrents from these sites or groups (comma-separated)")
	command.Flags().StringVarP(&excludeSites, "exclude-sites", "", "",
		"Do NOT add xseed torrents from these sites or groups (comma-separated)")
	command.Flags().StringVarP(&category, "category", "", "", constants.HELP_ARG_CATEGORY_XSEED)
	command.Flags().StringVarP(&tag, "tag", "", "", constants.HELP_ARG_TAG_XSEED)
	command.Flags().StringVarP(&filter, "filter", "", "", "Only xseed torrents which name contains this")
	command.Flags().StringVarP(&addCategory, "add-category", "", "",
		"Manually set category of added xseed torrent. By Default it uses the original torrent's")
	command.Flags().StringVarP(&addTags, "add-tags", "", "", "Set tags of added xseed torrent (comma-separated)")
	command.Flags().StringVarP(&minTorrentSizeStr, "min-torrent-size", "", "1GiB",
		"Torrents with size smaller than (<) this value will NOT be xseeded. -1 == no limit")
	command.Flags().StringVarP(&maxTorrentSizeStr, "max-torrent-size", "", "-1",
		"Torrents with size larger than (>) this value will NOT be xseeded. -1 == no limit")
	cmd.AddEnumFlagP(command, &iyuuRequestServer, "request-server", "",
		common.YesNoAutoFlag("Whether or not send request to iyuu server to update local xseed db"))
	iyuu.Command.AddCommand(command)
}

func xseed(cmd *cobra.Command, args []string) error {
	log.Tracef("iyuu token: %s", config.Get().IyuuToken)
	if config.Get().IyuuToken == "" {
		return fmt.Errorf("you must config iyuuToken in ptool.toml to use iyuu functions")
	}

	includeSitesMode := false
	includeSitesFlag := map[string]bool{}
	excludeSitesFlag := map[string]bool{}
	if includeSites != "" && excludeSites != "" {
		return fmt.Errorf("--include-sites and --exclude-sites flags can NOT be both set")
	}
	if includeSites != "" {
		includeSitesMode = true
		sites := config.ParseGroupAndOtherNames(util.SplitCsv(includeSites)...)
		for _, site := range sites {
			includeSitesFlag[site] = true
		}
	} else if excludeSites != "" {
		sites := config.ParseGroupAndOtherNames(util.SplitCsv(excludeSites)...)
		for _, site := range sites {
			excludeSitesFlag[site] = true
		}
	}
	minTorrentSize, _ := util.RAMInBytes(minTorrentSizeStr)
	maxTorrentSize, _ := util.RAMInBytes(maxTorrentSizeStr)
	filter = strings.ToLower(filter)
	var fixedTags []string
	if addTags != "" {
		fixedTags = util.SplitCsv(addTags)
	}

	clientNames := args
	clientInstanceMap := map[string]client.Client{} // clientName => clientInstance
	clientInfoHashesMap := map[string][]string{}
	reqInfoHashes := []string{}

	cntCandidateTargetTorrents := int64(0)
	cntTargetTorrents := int64(0)
	cntXseedTorrents := int64(0)
	cntSucccessXseedTorrents := int64(0)

	for _, clientName := range clientNames {
		clientInstance, err := client.CreateClient(clientName)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		clientInstanceMap[clientName] = clientInstance

		torrents, err := clientInstance.GetTorrents("", "", true)
		if err != nil {
			log.Errorf("client %s failed to get torrents: %v", clientName, err)
			continue
		} else {
			log.Tracef("client %s has %d torrents", clientName, len(torrents))
		}
		sort.Slice(torrents, func(i, j int) bool {
			if torrents[i].Size != torrents[j].Size {
				return torrents[i].Size > torrents[j].Size
			}
			a, b := 0, 0
			if torrents[i].HasTag(config.XSEED_TAG) {
				a = 1
			}
			if torrents[j].HasTag(config.XSEED_TAG) {
				b = 1
			}
			if a != b {
				return a < b
			}
			if torrents[i].Atime != torrents[j].Atime {
				return torrents[i].Atime < torrents[j].Atime
			}
			return torrents[i].TrackerDomain < torrents[j].TrackerDomain
		})
		infoHashes := []string{}
		tsize := int64(0)
		var sameSizeTorrentContentPathes []string
		for _, torrent := range torrents {
			// same size torrents may be identical
			if torrent.Size != tsize {
				sameSizeTorrentContentPathes = []string{torrent.ContentPath}
				reqInfoHashes = append(reqInfoHashes, torrent.InfoHash)
				tsize = torrent.Size
			} else if !slices.Contains(sameSizeTorrentContentPathes, torrent.ContentPath) {
				sameSizeTorrentContentPathes = append(sameSizeTorrentContentPathes, torrent.ContentPath)
				reqInfoHashes = append(reqInfoHashes, torrent.InfoHash)
			}
			if category != "" {
				if category == constants.NONE {
					if torrent.Category != "" {
						continue
					}
				} else if torrent.Category != category {
					continue
				}
			} else if strings.HasPrefix(torrent.Category, "_") {
				continue
			}
			if torrent.HasTag(config.NOXSEED_TAG) {
				continue
			}
			if tag != "" {
				if tag == constants.NONE {
					if len(torrent.Tags) > 0 {
						continue
					}
				} else if !torrent.HasAnyTag(tag) {
					continue
				}
			}
			if torrent.State != "seeding" || !torrent.IsFullComplete() ||
				(minTorrentSize >= 0 && torrent.Size < minTorrentSize) ||
				(maxTorrentSize >= 0 && torrent.Size > maxTorrentSize) {
				continue
			}
			if filter != "" && !strings.Contains(torrent.Name, filter) {
				continue
			}
			infoHashes = append(infoHashes, torrent.InfoHash)
			cntCandidateTargetTorrents++
		}
		clientInfoHashesMap[clientName] = infoHashes
	}

	if cntCandidateTargetTorrents == 0 {
		fmt.Printf("No cadidate torrents to to xseed.")
		return nil
	}

	reqInfoHashes = util.UniqueSlice(reqInfoHashes)
	doRequestServer := false
	if iyuuRequestServer == "auto" {
		var lastUpdateTime iyuu.Meta
		iyuu.Db().Where("key = ?", "lastUpdateTime").First(&lastUpdateTime)
		if lastUpdateTime.Value == "" || util.Now()-util.ParseInt(lastUpdateTime.Value) >= 7200 {
			doRequestServer = true
		} else {
			log.Tracef("Fetched iyuu xseed data recently. Do not fetch this time")
		}
	} else if iyuuRequestServer == "yes" {
		doRequestServer = true
	}
	if doRequestServer {
		updateIyuuDatabase(config.Get().IyuuToken, reqInfoHashes)
	}

	var sites []iyuu.Site
	var clientTorrents []*iyuu.Torrent
	var clientTorrentsMap = map[string][]*iyuu.Torrent{} // targetInfoHash => iyuuTorrent
	iyuu.Db().Find(&sites)
	iyuu.Db().Where("target_info_hash in ?", reqInfoHashes).Find(&clientTorrents)
	site2LocalMap := iyuu.GenerateIyuu2LocalSiteMap(sites, config.Get().SitesEnabled)
	log.Tracef("iyuu->ptool site map: %v; clientTorrents: len=%d", site2LocalMap, len(clientTorrents))
	for _, torrent := range clientTorrents {
		list := clientTorrentsMap[torrent.TargetInfoHash]
		list = append(list, torrent)
		clientTorrentsMap[torrent.TargetInfoHash] = list
	}

	siteInstancesMap := map[string]site.Site{}
	siteConsecutiveFails := map[string]int64{}
mainloop:
	for i, clientName := range clientNames {
		log.Printf("Start xseeding client (%d/%d) %s", i+1, len(clientName), clientName)
		clientInstance := clientInstanceMap[clientName]
		cnt := len(clientInfoHashesMap[clientName])
		for i, infoHash := range clientInfoHashesMap[clientName] {
			if i > 0 && slowMode {
				util.Sleep(3)
			}
			xseedTorrents := clientTorrentsMap[infoHash]
			if len(xseedTorrents) == 0 {
				log.Debugf("torrent %s skipped or has no xseed candidates", infoHash)
				continue
			} else {
				log.Debugf("torrent %s has %d xseed candidates", infoHash, len(xseedTorrents))
			}
			targetTorrent, err := clientInstance.GetTorrent(infoHash)
			if err != nil {
				log.Errorf("Failed to get target torrent %s info from client: %v", infoHash, err)
				continue
			}
			cntTargetTorrents++
			log.Tracef("client torrent (%d/%d) %s: name=%s, savePath=%s",
				i+1, cnt,
				targetTorrent.InfoHash, targetTorrent.Name, targetTorrent.SavePath,
			)
			targetTorrentContentFiles, err := clientInstance.GetTorrentContents(infoHash)
			if err != nil {
				log.Tracef("Failed to get target torrent %s contents from client: %v", infoHash, err)
				continue
			}
			for _, xseedTorrent := range xseedTorrents {
				clientExistingTorrent, err := clientInstance.GetTorrent(xseedTorrent.InfoHash)
				if err != nil {
					log.Errorf("Failed to get client existing torrent info for %s", xseedTorrent.InfoHash)
					continue
				}
				sitename := site2LocalMap[xseedTorrent.Sid]
				if sitename == "" {
					log.Tracef("torrent %s xseed candidate torrent %s site sid %d not found in local",
						infoHash, xseedTorrent.InfoHash, xseedTorrent.Sid,
					)
					continue
				}
				if maxConsecutiveFail >= 0 && siteConsecutiveFails[sitename] > maxConsecutiveFail {
					log.Debugf("Skip site %s torrent %s as this site has failed too much times", sitename, xseedTorrent.InfoHash)
					continue
				}
				if clientExistingTorrent != nil {
					log.Tracef("xseed candidate %s already existed in client", xseedTorrent.InfoHash)
					if !dryRun {
						tags := []string{}
						removeTags := []string{}
						if !clientExistingTorrent.HasTag(config.XSEED_TAG) {
							tags = append(tags, config.XSEED_TAG)
						}
						siteTag := client.GenerateTorrentTagFromSite(sitename)
						if !clientExistingTorrent.HasTag(siteTag) {
							tags = append(tags, siteTag)
						}
						oldSite := clientExistingTorrent.GetSiteFromTag()
						if oldSite != "" && oldSite != sitename {
							removeTags = append(removeTags, client.GenerateTorrentTagFromSite(oldSite))
						}
						if len(tags) > 0 || len(removeTags) > 0 {
							clientInstance.ModifyTorrent(clientExistingTorrent.InfoHash, &client.TorrentOption{
								Tags:       tags,
								RemoveTags: removeTags,
							}, nil)
						}
					}
					continue
				}
				if (includeSitesMode && !includeSitesFlag[sitename]) || (!includeSitesMode && excludeSitesFlag[sitename]) {
					log.Tracef("skip site %s torrent", sitename)
					continue
				}
				if siteInstancesMap[sitename] == nil {
					siteInstance, err := site.CreateSite(sitename)
					if err != nil {
						return fmt.Errorf("failed to create iyuu sid %d (local %s) site instance: %w",
							xseedTorrent.Sid, sitename, err)
					}
					siteInstancesMap[sitename] = siteInstance
				}
				siteInstance := siteInstancesMap[sitename]
				log.Printf("Xseed torrent %s (target %s) from site %s (iyuu sid %d) / tid %d",
					xseedTorrent.InfoHash,
					targetTorrent.Name,
					sitename,
					xseedTorrent.Sid,
					xseedTorrent.Tid,
				)
				if dryRun {
					continue
				}
				xseedTorrentContent, _, err := siteInstance.DownloadTorrentById(fmt.Sprint(xseedTorrent.Tid))
				if err != nil {
					log.Errorf("Failed to download torrent from site: %v", err)
					if !strings.Contains(err.Error(), "status=404") {
						siteConsecutiveFails[sitename]++
						if maxConsecutiveFail >= 0 && siteConsecutiveFails[sitename] == maxConsecutiveFail {
							log.Errorf("Site %s has consecutively failed (to download torrent) too many times, skip it from now",
								sitename)
						}
					} else {
						siteConsecutiveFails[sitename] = 0
					}
					continue
				}
				siteConsecutiveFails[sitename] = 0
				xseedTorrentInfo, err := torrentutil.ParseTorrent(xseedTorrentContent)
				if err != nil {
					log.Errorf("Failed to parse xseed torrent contents: %v", err)
					continue
				}
				compareResult := xseedTorrentInfo.XseedCheckWithClientTorrent(targetTorrentContentFiles)
				if compareResult < 0 {
					if compareResult == -2 {
						log.Tracef("xseed candidate is NOT identital with client torrent. (Only ROOT folders diff)")
					} else {
						log.Tracef("xseed candidate is NOT identital with client torrent.")
					}
					continue
				}
				cntXseedTorrents++
				xseedTorrentCategory := targetTorrent.Category
				if addCategory != "" {
					xseedTorrentCategory = addCategory
				}
				tags := []string{config.XSEED_TAG, client.GenerateTorrentTagFromSite(sitename)}
				tags = append(tags, fixedTags...)
				ratioLimit := float64(0)
				if xseedTorrentInfo.IsPrivate() {
					tags = append(tags, config.PRIVATE_TAG)
				} else {
					tags = append(tags, config.PUBLIC_TAG)
					ratioLimit = config.Get().PublicTorrentRatioLimit
				}
				err = clientInstance.AddTorrent(xseedTorrentContent, &client.TorrentOption{
					SavePath:     targetTorrent.SavePath,
					Category:     xseedTorrentCategory,
					Tags:         tags,
					Pause:        addPaused,
					SkipChecking: !check,
					RatioLimit:   ratioLimit,
				}, nil)
				log.Infof("Add xseed torrent %s result: error=%v", xseedTorrent.InfoHash, err)
				if err == nil {
					cntSucccessXseedTorrents++
				}
				if maxXseedTorrents >= 0 && cntXseedTorrents >= maxXseedTorrents {
					break mainloop
				}
			}
		}
	}
	fmt.Printf("Done xseed %d clients. Target / Xseed / SuccessXseed torrents: %d / %d / %d\n",
		len(clientNames), cntTargetTorrents, cntXseedTorrents, cntSucccessXseedTorrents)
	return nil
}

func updateIyuuDatabase(token string, allInfoHashes []string) error {
	log.Debugf("Querying iyuu server for xseed info of %d torrents.", len(allInfoHashes))

	// update sites
	iyuuSites, err := iyuu.IyuuApiSites(token)
	if err != nil {
		log.Errorf("failed to get iyuu sites: %v", err)
	} else {
		iyuu.Db().Transaction(func(tx *gorm.DB) error {
			tx.Where("1 = 1").Delete(&iyuu.Site{})
			iyuuSiteRecords := util.Map(iyuuSites, func(iyuuSite *iyuu.IyuuApiSite) iyuu.Site {
				return iyuu.Site{
					Sid:          iyuuSite.Id,
					Name:         iyuuSite.Site,
					Nickname:     iyuuSite.Nickname,
					Url:          iyuuSite.GetUrl(),
					DownloadPage: iyuuSite.Download_page,
				}
			})
			tx.Create(&iyuuSiteRecords)
			return nil
		})
	}

	// report existing sites
	sid_sha1, err := iyuu.IyuuApiReportExisting(token, iyuuSites)
	if err != nil {
		log.Errorf("failed to report existing sites: %v", err)
	}

	for len(allInfoHashes) > 0 {
		number := min(len(allInfoHashes), iyuu.MAX_INTOHASH_NUMBER)
		infoHashes := allInfoHashes[:number]
		allInfoHashes = allInfoHashes[number:]

		// update xseed torrents data
		data, err := iyuu.IyuuApiHash(token, infoHashes, sid_sha1)
		if err != nil {
			log.Errorf("iyuu apiHash error: %v", err)
		} else {
			log.Debugf("iyuu data len(data)=%d\n", len(data))
			iyuu.Db().Transaction(func(tx *gorm.DB) error {
				for targetInfoHash, iyuuRecords := range data {
					tx.Where("target_info_hash = ?", targetInfoHash).Delete(&iyuu.Torrent{})
					infoHashes := util.Map(iyuuRecords, func(record iyuu.IyuuTorrentInfoHash) string {
						return record.Info_hash
					})
					tx.Where("info_hash in ?", infoHashes).Delete(&iyuu.Torrent{})
					iyuuTorrents := util.Map(iyuuRecords, func(iyuuRecord iyuu.IyuuTorrentInfoHash) iyuu.Torrent {
						return iyuu.Torrent{
							InfoHash:       iyuuRecord.Info_hash,
							Sid:            iyuuRecord.Sid,
							Tid:            iyuuRecord.Torrent_id,
							TargetInfoHash: targetInfoHash,
						}
					})
					tx.Create(&iyuuTorrents)
				}

				tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "key"}},
					DoUpdates: clause.AssignmentColumns([]string{"value"}),
				}).Create(&iyuu.Meta{
					Key:   "lastUpdateTime",
					Value: fmt.Sprint(util.Now()),
				})
				return nil
			})
		}
	}

	return nil
}
