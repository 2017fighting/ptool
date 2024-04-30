// golang do NOT allow non-primitive constants so some values are defined as variables.
// However, all variables in this package never change in runtime.
package constants

import (
	"regexp"
)

// 如果 ptool.toml 配置文件里字符串类型配置项值为空，使用系统默认值；使用 NONE 值显式设置该配置项为空值。
// 部分 flag 参数使用 NONE 值显式指定为空值。
const NONE = "none"

// a special proxy value to indicate force use proxy from HTTP(S)_PROXY env.
const ENV_PROXY = "env"

const FILENAME_INVALID_CHARS_REGEX = `[<>:"/\|\?\*]+`
const PERM = 0600 // 程序创建的所有文件的 PERM

const FILENAME_SUFFIX_ADDED = ".added"
const FILENAME_SUFFIX_OK = ".ok"
const FILENAME_SUFFIX_FAIL = ".fail"
const FILENAME_SUFFIX_BACKUP = ".bak"

// Some funcs require a (positive) timeout parameter. Use a very long value to emulate infinite. (Seconds)
const INFINITE_TIMEOUT = 86400 * 365 * 100

const BIG_FILE_SIZE = 10 * 1024 * 1024 // 10MiB
const FILE_HEADER_CHUNK_SIZE = 512
const INFINITE_SIZE = 1024 * 1024 * 1024 * 1024 * 1024 * 1024 // 1EiB

const CLIENT_DEFAULT_DOWNLOADING_SPEED_LIMIT = 300 * 1024 * 1024 / 8 // BT客户端默认下载速度上限：300Mbps

// type, name, ↑info, ↓info, others
const STATUS_FMT = "%-6s  %-15s  %-27s  %-27s  %-s\n"

var FilenameInvalidCharsRegex = regexp.MustCompile(FILENAME_INVALID_CHARS_REGEX)

// .torrent file magic number.
// See: https://en.wikipedia.org/wiki/Torrent_file , https://en.wikipedia.org/wiki/Bencode .
// 大部分 .torrent 文件第一个字段是 announce，
// 个别种子没有 announce / announce-list 字段，第一个字段是 created by / creation date 等，
// 这类种子可以通过 DHT 下载成功。
// values: ["d8:announce", "d10:created by", "d13:creation date"]
var TorrentFileMagicNumbers = []string{"d8:announce", "d13:announce-list", "d10:created by", "d13:creation date"}

// Some ptool cmds could add a suffix to processed (torrent) filenames.
// Current Values: [".added", ".ok", ".fail", ".bak"].
var ProcessedFilenameSuffixes = []string{
	FILENAME_SUFFIX_ADDED,
	FILENAME_SUFFIX_OK,
	FILENAME_SUFFIX_FAIL,
	FILENAME_SUFFIX_BACKUP,
}

// Sources:
// https://github.com/nyaadevs/nyaa/blob/master/trackers.txt ,
// https://github.com/ngosang/trackerslist/blob/master/trackers_best.txt .
// Only include most popular & stable trackers in this list.
var OpenTrackers = []string{
	"udp://open.stealth.si:80/announce",
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://exodus.desync.com:6969/announce",
	"udp://open.demonii.com:1337/announce",      // At least since 2014
	"udp://tracker.torrent.eu.org:451/announce", // Since 2016: https://github.com/ngosang/trackerslist/issues/26
	// Runned by Internet Archive.
	// According to https://help.archive.org/help/archive-bittorrents/,
	//  they are not open ("they track our only own torrents").
	// "udp://bt1.archive.org:6969/announce",
	// "udp://bt1.archive.org:6969/announce",
	"http://sukebei.tracker.wf:8888/announce", // nyaa
	"http://nyaa.tracker.wf:7777/announce",
}
