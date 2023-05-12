# ptool

自用的 PT (private tracker) 网站辅助工具。提供全自动刷流(brush)、自动辅种(使用 iyuu 接口)、BT客户端控制等功能。

主要特性：

* 使用 Go 开发的纯 CLI 程序。单文件可执行程序，没有外部依赖。支持 Windows / Linux、x64 / arm64 等多种环境、架构。
* 无状态(stateless)：程序自身不保存任何状态、不在后台持续运行。“刷流”等任务需要使用 cron job 等方式定时运行本程序。
* 使用简单。只需5分钟时间，配置 BT 客户端地址、PT 网站地址和 cookie 即可开始全自动刷流。
* 目前支持的 BT 客户端： qBittorrent 4.1+。
* 目前支持的 PT 站点：绝大部分使用 nexusphp 的网站。
  * 测试过支持的站点：M-Team(馒头)、柠檬、U2、冬樱、红叶、聆音、铂金家、若干不可说的站点。
  * 未列出的大部分 np 站点应该也支持。除了个别魔改 np 很厉害的站点可能不支持。
* 刷流功能(brush)：
  * 不依赖 RSS。直接抓取站点页面上最新的种子。
  * 无需配置选种规则。自动跳过非免费的和有 HR 的种子；自动筛选适合刷流的种子。
  * 无需配置删种规则。自动删除已无刷流价值的种子；自动删除免费时间到期的种子；硬盘空间不足时也会自动删种。

## 快速开始（刷流）

下载本程序的可执行文件 ptool (Linux) 或 ptool.exe (Windows) 放到任意目录，在同目录下创建名为 "ptool.toml" 的配置文件，内容示例如下：

```toml
[[clients]]
name = "local"
type = "qbittorrent" # 客户端类型。目前只支持 qbittorrent v4.1+。需要启用 Web UI。
url = "http://localhost:8080/" # qBittorrent web UI 地址
username = "admin" # QB Web UI 用户名
password = "adminadmin" # QB Web UI 密码

[[sites]]
type = "mteam"
cookie = "cookie_here" # 浏览器 F12 获取的网站 cookie
```

然后在当前目录下运行 ```ptool brush local mteam``` 即可执行刷流任务。程序会从 M-Team 获取最新的种子、根据一定规则筛选出适合的种子添加到本地的 qBittorrent 客户端里，同时自动从 BT 客户端里删除（已经没有上传的）旧的刷流种子。刷流任务添加到客户端里的种子会放到 ```_brush``` 分类(Category)里。程序只会对这个分类里的种子进行管理或删除等操作。

使用 cron job / 计划任务等方式定时执行上面的刷流任务命令（例如每隔 10 分钟执行一次）即可。

## 配置文件

程序支持使用 toml 或 yaml 格式的配置文件（```ptool.toml``` 或 ```ptool.yaml```），推荐使用前者。

配置文件可以直接放到程序启动时的当前目录(cwd)下，也可以选择放到当前操作系统用户主目录下的 ".config/ptool/" 路径下（推荐）：

* Linux: ```~/.config/ptool/ptool.toml```
* Windows: ```%USERPROFILE%\.config\ptool\ptool.toml```

也可以通过启动程序时传入命令行参数 ```--config ptool.toml``` 手动指定使用的配置文件路径。

配置文件里可以使用 ```[[clients]]``` 和 ```[[sites]]``` 区块添加任意多个 BT 客户端和站点。

```[[site]]``` 区块有两种配置方式：

```toml
# 方式 1（推荐）：直接使用站点 ID 或 alias 作为类型(type)。无需手动输入站点 url。站点名称(name)默认使用 type。
[[sites]]
type = "mteam"
cookie = "cookie_here" # 浏览器 F12 获取的网站 cookie

# 方式 2：使用通用的 nexusphp 类型，需要手动指定站点名称(name)、站点 url 和其他参数。
[[sites]]
name = "mteam"
type = "nexusphp" # 站点类型。目前只支持 nexusphp
url = "https://kp.m-team.cc/" # 站点首页 URL
cookie = "cookie_here" # 浏览器 F12 获取的网站 cookie
```

推荐使用“方式 1”。程序内置了对大部分国内 NexusPHP PT 站点的支持。站点 ID 通常为 PT 网站域名的主体部分（不含次级域名和 TLD 部分），例如 BTSCHOOL ( https://pt.btschool.club/ )的站点 ID 是 btschool。部分 PT 网站也可以使用别名(alias)配置，例如 M-TEAM ( https://kp.m-team.cc/ )在本程序配置文件里的 type 设为 "m-team" 或 "mteam" 均可。运行 ```ptool sites``` 查看所有本程序内置支持的 PT 站点列表。本程序没有内置支持的 PT 站点必须通过“方式 2”配置。 

参考程序代码根目录下的 ```ptool.example.toml``` 和 ```ptool.example.yaml``` 示例配置文件了解所有可用的配置项。

配置好站点后，使用 ```ptool status <site> -t``` 测试。如果配置正确且 Cookie 有效，会显示站点当前登录用户的状态信息和网站最新种子列表。

## 程序功能

所有功能通过启动程序时传入的第一个”命令“参数区分：

```
ptool <command> args...
```

所有可用的 &lt;command&gt; 包括:

* brush : 自动刷流。
* iyuu : 使用 iyuu 接口自动辅种。
* status : 显示 BT 客户端或 PT 站点当前状态信息。
* stats : 显示刷流任务流量统计。
* search : 在某个站点搜索指定关键词的种子
* add : 将某个站点的指定种子添加到 BT 客户端。
* dltorrent : 下载站点的种子。
* addlocal : 将本地的种子文件添加到 BT 客户端。
* BT 客户端控制命令集: clientctl / show / pause / resume / delete / reannounce / recheck / getcategories / setcategory / gettags / createtags / deletetags / addtags / removetags / edittracker / setsavepath
* parsetorrent : 显示种子(torrent)文件信息
* sites : 显示本程序内置支持的所有 PT 站点列表。
* version : 显示本程序版本信息。

运行 ```ptool``` 查看程序支持的所有命令列表；运行 ```ptool <command> -h``` 查看指定命令的参数格式和使用说明。

### 刷流 (brush)

```
ptool brush <client> <site>... [flags]
```

刷流任务从指定的站点获取最新种子，选择适当的种子加入 BT 客户端；并自动从客户端中删除旧的（已没有上传速度的）刷流任务种子及其文件。刷流任务的目标是使 BT 客户端的上传速度达到软件中设置的上传速度上限（如果客户端里没有设置上传速度上限，本程序默认使用 10MB/s 这个值），如果当前 BT 客户端的上传速度已经达到或接近了上限（不管上传是否来源于刷流任务添加的种子），程序不会添加任何新种子。

参数

* &lt;client&gt; : 配置文件里定义的 BT 客户端 name。
* &lt;site&gt; : 配置文件里定义的 PT 站点 name。

可以提供多个 &lt;site&gt; 参数。程序会按随机顺序从提供的 &lt;site&gt; 列表里的各站点获取最新种子、筛选并选取一定数量的合适的种子添加到 BT 客户端。可以将同一个站点名重复出现多次以增加其权重，使刷流任务添加该站点种子的几率更大。

示例

```
# 使用 local 这个 BT 客户端，刷流 mteam 站点
ptool brush local mteam
```

选种（选择新种子添加到 BT 客户端）规则：

* 不会选择有以下任意特征的种子：不免费、存在 HnR 考查、免费时间临近截止。
* 发布时间过久的种子也不会被选择。
* 种子的当前做种、下载人数，种子大小等因素也都会考虑。

删种（删除 BT 客户端里旧的刷流种子）规则：

* 未下载完成的种子免费时间临近截止时，删除种子或停止下载（只上传模式）。
* 硬盘剩余可用空间不足（默认保留 5GB）时，开始删除没有上传速度的种子。
* 未下载完成的种子，如果长时间没有上传速度或上传/下载速度比例过低，也可能被删除。


刷流任务添加到客户端里的种子会放到 ```_brush``` 分类里。程序只会对这个分类里的种子进行管理或删除等操作。不会干扰 BT 客户端里其它正常的下载任务。如果需要永久保留某个刷流任务添加的种子（防止其被自动删除），在 BT 客户端里更改其分类即可。

### 自动辅种 (iyuu)

iyuu 命令通过 [iyuu 接口](https://api.iyuu.cn/docs.php) 提供自动辅种(cross seed)功能。

#### iyuu 配置

如果是第一次使用 iyuu，首先需要在 [iyuu 网站](https://iyuu.cn/) 上微信扫码申请IYUU令牌（token）。在本程序的配置文件 ptool.toml 里配置 iyuu token：

```
iyuuToken = "IYUU0011223344..."
```

然后使用 iyuu 支持的任意合作站点的 uid 和 passkey 激活和绑定(bind) iyuu token，命令格式如下：

```
ptool iyuu bind --site zhuque --uid 123456 --passkey 0123456789abcdef
```

所有参数均必须提供

* --site : 用于验证的 PT 站点名。可以使用 `ptool iyuu sites -b` 命令查询 iyuu 支持的合作站点列表。
* --uid : 对应 PT 站点的用户 uid（数字）。在 PT 网站的个人页面获取。
* --passkey : 对应 PT 站点的用户 passkey。在 PT 网站的个人页面获取。

使用 ```ptool iyuu status``` 查询当前 iyuu token 的激活和绑定状态。

#### 使用 iyuu 自动辅种

```
ptool iyuu xseed <client>...
```

可以提供多个 client。程序会获取这些 client 里正在做种的种子信息，通过 iyuu 接口查询可以辅种的种子并将其自动添加到对应客户端里。注意只有在本程序的 ptool.toml 配置文件里添加的站点才会被辅种。

iyuu xseed 子命令支持很多可选参数。运行 ```ptool iyuu xseed -h``` 查看所有可选参数使用说明。

### BT 客户端控制命令集

提供了一系列管理、控制 BT 客户端的命令。

#### 读取/修改 BT 客户端配置 (clientctl)

```
ptool clientctl <client> [<option>[=value] ...]
```

clientctl 命令可以显示或修改指定 name 的 BT 客户端的配置参数。


支持的参数(&lt;option&gt;) 列表：

* global_download_speed_limit : 全局下载速度上限。
* global_upload_speed_limit : 全局上传速度上限。

示例：

```
# 获取 local 客户端所有参数当前值
ptool clientctl local

# 设置 local 客户端的全局上传速度限制为 10MB/s
ptool clientctl local global_upload_speed_limit=10M
```

#### 显示信息 / 暂停 / 恢复 / 删除 / 强制汇报 / 强制检测Hash 客户端里种子 (show / pause / resume / delete / reannounce / recheck)

命令格式均为：

```
ptool <command> <infoHash>...
```

&lt;infoHash&gt;参数为指定的 BT 客户端里需要操作的种子的 infoHash 列表。也可以使用以下特殊值参数操作多个种子：

* _all : 所有种子
* _done : 所有已完成的种子（无论是否正在做种）
* _active : 当前正在活动（上传或下载）的种子
* _error : 状态为“出错”的种子
* _downloading / _seeding / _paused / _complete : 状态为正在下载 / 做种 / 暂停下载 / 已完成的种子

示例：

```
# 强制立即汇报所有种子
ptool reannounce local _all

# 恢复下载/做种所有种子
ptool resume local _all

# 从客户端删除指定种子（默认同时删除文件）
ptool delete local 31a615d5984cb63c6f999f72bb3961dce49c194a

# 特别的，如果 show 命令只提供一个 infoHash 参数，会显示该种子的所有详细信息。
ptool show local 31a615d5984cb63c6f999f72bb3961dce49c194a
```

#### 管理 BT 客户端里的的种子分类 / 标签 (getcategories / setcategory / gettags / createtags / deletetags / addtags / removetags / edittracker / setsavepath)

```
# 获取所有分类
ptool getcategories <client>

# 修改种子的所属分类
ptool setcategory <client> <category> <infoHashes>...

# 获取所有标签(tag)
ptool gettags <client>

# 创建新的标签
ptool createtags <client> <tags>...

# 删除标签
ptool deletetags <client> <tags>...

# 为客户端里种子添加tag
ptool addtags <client> <tags> <infoHashes>...

# 为客户端里种子删除tag
ptool removetags <client> <tags> <infoHashes>...

# 修改种子的 tracker
ptool edittracker <client> <infoHashes...> --old-tracker "https://..." --new-tracker "https://..."

# 修改种子内容的保存路径
ptool setsavepath <client> <savePath> [<infoHash>...]
```

### 显示 BT 客户端或 PT 站点状态 (status)

```
ptool status <clientOrSite>...
```

显示指定 name 的 BT 客户端或 PT 站点的当前状态信息。可以提供多个名称。

显示的信息包括：

* BT 客户端：显示当前下载 / 上传速度和其上限，硬盘剩余可用空间。
* PT 站点：显示用户名、上传量、下载量。

可选参数：

* -t : 显示 BT 客户端或站点的种子列表（BT 客户端：当前活动的种子；PT 站点：最新种子）。
* -f : 显示完整的种子列表信息。

### 显示刷流任务流量统计 (stats)

```
ptool stats [client...]
```

显示 BT 客户端的刷流任务流量统计信息（下载流量、上传流量总和）。本功能默认不启用，如需启用，在 ptool.toml 配置文件的最上方里增加一行：```brushEnableStats: true``` 配置项。启用刷流统计后，刷流任务会使用 ptool.toml 配置文件相同目录下的 "ptool_stats.txt" 文件存储所需保存的信息。

只有刷流任务添加和管理的 BT 客户端的种子（即 ```_brush``` 分类的种子）的流量信息会被记录和统计。目前设计只有在刷流任务从 BT 客户端删除某个种子时才会记录和统计该种子产生的流量信息。

### 添加站点种子到 BT 客户端 (add)

```
ptool add <client> <torrentIdOrUrl>...
```

示例：

```
ptool add local mteam.488424
ptool add local --site mteam 488424
ptool add local "https://kp.m-team.cc/details.php?id=488424"
ptool add local "https://kp.m-team.cc/download.php?id=488424"
```

以上几条命令均可以将 M-Team 站点上ID为 [488424](https://kp.m-team.cc/details.php?id=488424&hit=1)  的种子添加到 "local" BT客户端。

### 下载站点的种子

```
ptool dltorrent <torrentIdOrUrl>...
```

类似 add 命令，但只会将种子下载到本地。

参数：

* --download-dir : 下载的种子文件保存路径。默认为当前目录(CWD)。

### 添加本地种子到 BT 客户端 (addlocal)

```
ptool addlocal <client> <filename.torrent>...
```

将本地硬盘里的种子文件添加的 BT 客户端。


### 搜索 PT 站点种子 (search)

```
ptool search <sites> <keyword>
```

&lt;sites&gt; 参数为需要所搜索的 PT 站点，可以使用 "," 分割提供多个站点。可以使用 "_all" 搜索所有已配置的 PT 站点。

可以用 ```ptool add``` 命令将搜索结果列表中的种子添加到 BT 客户端。

### 批量下载种子 (batchdl)

提供一个 batchdl 命令用于批量下载 PT 网站里体积最小的种子做种。（别名：ebookgod）

```
# 默认显示 PT 站点上找到的体积最小的 100 个种子列表
ptool batchdl <site>

# 下载找到的种子到当前目录
ptool batchdl <site> --action download


# 直接将种子添加到 "local" BT 客户端里
ptool batchdl <site> --action add --add-client local
```

参数：
* -m int : 最多下载多少个种子。默认 0（无限制，一直运行除非手动 Ctrl + C 停止）.
* --sort string : 站点种子排序方式：size|time|name|seeders|leechers|snatched|none (default size)
* --order string : 排序顺序：asc|desc。默认 asc。
* --min-torrent-size string : 种子大小的最小值限制。默认为 0。
* --max-torrent-size string : 种子大小的最大值限制。默认为 "0"（无限制）。
* --free : 只下载免费种子。
* --base-url : 手动指定种子列表页 URL，例如 "special.php"。

### 显示种子文件信息 (parsetorrent)

```
ptool parsetorrent file.torrent...
```

显示本地硬盘里的种子文件的元信息。

### 站点分组 (group) 功能

在 ptool.toml 配置文件里可以定义站点分组，例如：

```
[[groups]]
name = "acg"
sites = ["u2", "kamept"]
```

定义分组后，大部分命令中 &lt;site&gt; 类型的参数可以使用分组名代替以指代多个站点，例如：

```
# 在 acg 分组的所有站点中搜索 "clannad" 关键词的种子
ptool search acg clannad
```