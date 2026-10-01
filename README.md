<p align="center">
  <img width="240" src="https://github.com/yliu7949/KouShare-dl/blob/main/logo.png" style="text-align: center;" alt="KouShare-dl logo">
</p>

# KouShare-dl

[![License](https://img.shields.io/github/license/yliu7949/KouShare-dl.svg)](https://github.com/yliu7949/KouShare-dl/blob/main/LICENSE)
[![Build Status](https://github.com/yliu7949/KouShare-dl/workflows/Go/badge.svg)](https://github.com/yliu7949/KouShare-dl/actions?query=workflow%3AGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/yliu7949/KouShare-dl)](https://goreportcard.com/report/github.com/yliu7949/KouShare-dl)
[![Github Downloads](https://img.shields.io/github/downloads/yliu7949/KouShare-dl/total.svg)](http://gra.caldis.me/?url=https://github.com/yliu7949/KouShare-dl)
<a title="Hits" target="_blank" href="https://github.com/yliu7949/KouShare-dl"><img src="https://hits.b3log.org/yliu7949/KouShare-dl.svg"></a>
[![Github Release Version](https://img.shields.io/github/v/release/yliu7949/KouShare-dl?color=green&include_prereleases)](https://github.com/yliu7949/KouShare-dl/releases/latest)


KouShare-dl 是一个使用 [Cobra](https://github.com/spf13/cobra)
开发的用于从 [“蔻享学术”](https://www.koushare.com/) 视频网站下载视频和课件的命令行工具。

您可以在常见的操作系统（Windows，macOS 和 Linux 等）里使用该命令行工具。该工具已被发布至公有领域，因此您可以按照您的想法自由使用它，如对它进行修改、重新发布等操作。

# 目录

- [功能](#功能)
    + [它目前具有如下功能](#它目前具有如下功能)
    + [它无法做到的事情](#它无法做到的事情)
    + ["功能支持"表格](#功能支持表格)
- [编译](#编译)
- [使用方法](#使用方法)
- [命令简介](#命令简介)
- [示例](#示例)
  * [一、登录账户与注销登陆](#一登录账户与注销登陆)
    + [1.1 登录蔻享账户](#11-登录蔻享账户)
    + [1.2 注销登录状态](#12-注销登录状态)
  * [二、查看视频或直播信息](#二查看视频或直播信息)
  * [三、下载视频](#三下载视频)
    + [3.1 使用默认参数下载视频](#31-使用默认参数下载视频)
    + [3.2 下载视频至指定文件夹](#32-下载视频至指定文件夹)
    + [3.3 下载某个专题的所有视频](#33-下载某个专题的所有视频)
    + [3.4 下载不同清晰度的视频](#34-下载不同清晰度的视频)
    + [3.5 批量下载指定的视频](#35-批量下载指定的视频)
  * [四、录制直播与下载快速回放](#四录制直播与下载快速回放)
    + [4.1 对指定直播间进行录制](#41-对指定直播间进行录制)
    + [4.2 合并录制的视频片段](#42-合并录制的视频片段)
    + [4.3 下载直播间快速回放视频](#43-下载直播间快速回放视频)
  * [五、下载课件](#五下载课件)
    + [5.1 下载单个课件和专题课件](#51-下载单个课件和专题课件)
    + [5.2 优化 pdf 文件【实验性功能】](#52-优化-pdf-文件实验性功能)
  * [六、清理临时文件](#六清理临时文件)
- [FAQ](#faq)
    - [KouShare-dl 下载视频时是并行下载吗？](#koushare-dl-下载视频时是并行下载吗)
    - [下载专题视频时因网络波动导致下载中断该怎么办？](#下载专题视频时因网络波动导致下载中断该怎么办)
    - [下载视频的过程中遇到因被占用而导致文件重命名失败的错误应该如何处理？](#下载视频的过程中遇到因被占用而导致文件重命名失败的错误应该如何处理)
- [鸣谢](#鸣谢)
- [许可证合规性](#许可证合规性)

# 功能

### 它目前具有如下功能

- 通过网页登录或短信验证码登录蔻享账户

- 获取视频或直播的详细信息

- 下载单个蔻享视频或整个专题的视频

- 下载清晰度为标清、高清和超清的视频（需要登录）

- 下载**已购买且在有效期内**的付费视频（需要登录）

- 批量下载指定的视频

- 定时录制直播间

- 下载已提供标准回放地址或正式视频 ID 的直播回放

- 下载单个课件或整个专题的课件

### 它无法做到的事情

- 下载未购买的付费视频

- 绕过网站的人机验证、付费或其他访问控制。公开内容不需要先访问首页；受限内容请在浏览器正常完成验证和登录后导入自己的令牌。

### "功能支持"表格

| 类型 | 是否支持专题下载 | 是否支持单独下载 | 是否支持断点续传 | 是否支持不同清晰度的下载 | 是否支持付费产品下载 |
| :--: | :--------------: | :--------------: | :--------------: | :----------------------: | :------------------: |
| 视频 |        ✔️         |        ✔️         |        ❌         |            ✔️             |          ⭕           |
| 直播 |        ➖         |        ✔️         |        ❌         |            ✔️             |          ➖           |
| 课件 |        ✔️         |        ✔️         |        ❌         |            ➖             |          ✔️           |

（✔️表示支持该功能，❌表示不支持该功能，➖表示该功能不存在，⭕表示部分支持该功能）

# 编译

您可以下载 [Releases](https://github.com/yliu7949/KouShare-dl/releases/latest)
中的二进制文件`ks.exe`或`ks`后直接使用，也可以下载源代码自行编译。

从源码编译需要 Go 1.27.1 或更新的兼容版本。
### Windows
```shell
go build -o ks.exe -trimpath -ldflags "-s -w -buildid=" ks.go
```

### Linux
```shell
go build -o ks -trimpath -ldflags "-s -w -buildid=" ks.go
```

# 使用方法

您需要通过命令行或终端进入该程序所在的文件夹，才能执行相关命令。

以`Windows`平台为例，若可执行程序`ks.exe`位于`C:\Users\lenovo\Downloads\`路径下，您每次使用时需要通过快捷键`Win`+`R`打开“运行”对话框，输入`CMD`后回车打开命令行窗口。在命令行窗口中输入以下命令：

```shell
cd C:\Users\lenovo\Downloads\
ks version
```

若出现`KouShare-dl v0.9.1`字样，则说明可以正常使用。接下来您可以继续输入 KouShare-dl 程序的命令来进行交互。比如，输入`ks help`并回车，您就可以看到 KouShare-dl 程序的帮助信息了。

# 命令简介

KouShare-dl 程序的命令具有下面的格式：

```shell
  ks [command] <flag>
```

其中`[command]`为必选命令，`<flag>`为可选参数。

可使用的 command 命令：

```shell
  clean       清除指定目录下的所有tmp临时文件
  help        查看某个具体命令的更多帮助信息
  info        获取视频或直播的基本信息
  login       打开浏览器登录并自动保存凭证
  logout      退出登陆
  merge       合并下载的视频片段文件
  record      录制指定直播间ID的直播，命令别名为live
  save        保存指定vid的视频（vid为视频网址里最后面的一串数字），命令别名为video
  slide       下载指定vid的视频对应的课件
  upgrade     升级为最新版本
  version     输出版本号，并检查最新版本
```

可使用的 flag 参数：

```shell
  -@, --at          指定时间，格式为"2006-01-02 15:04:05"
  -a, --autoMerge   指定是否自动合并下载的视频片段文件
  -c, --concurrency 指定单个视频同时下载的 HLS 片段数（默认 12）
  -h, --help        查看帮助信息
  -n, --name        指定输出文件的名字
      --password    指定直播间密码
  -p, --path        指定保存文件的路径（若不指定，则默认为该程序当前所在的路径）
  -p, --path        指定清理临时文件的路径（若不指定，则默认为该程序当前所在的路径）
  -P, --proxy       指定使用的http/https/socks5代理服务地址
  -q, --quality     指定下载视频的清晰度（high为超清，standard为高清，low为标清，不指定则默认为超清）
  -q, --quiet       指定是否不输出清理过程中的信息
      --qpdf-bin    指定qpdf的bin文件夹所在的路径（注：该flag无简写形式）
  -r, --replay      指定是否下载直播间快速回放视频
  -s, --series      指定是否下载整个专题的文件
      --nocolor     指定是否不使用彩色输出
      --video-concurrency 指定批量模式同时下载的视频数（默认 3）
  -v, --version     查看版本号
  -v, --vidPrefix   指定是否使用vid作为保存视频文件名的前缀
```

需要注意的是，对于每个 command 命令，仅有部分 flag 参数是可用且有效的。可以通过`ks help [command]`来查看某个命令的详细描述及其可用的 flag 参数。

# 示例

## 一、登录账户与注销登陆

登录不是查看公开信息和录制公开直播的必要条件。视频播放接口可能要求登录；付费内容还必须由当前账户合法购买且处于有效期内。

### 1.1 登录蔻享账户

网页登录：

```shell
ks login
```

命令会打开一个专用于 KouShare-dl 的 Chrome 配置并跳转到蔻享登录页。它不再启用 Chrome 的“自动测试软件”标志，并会保留用户亲自完成的 EdgeOne 验证 Cookie，避免每次运行都以全新浏览器身份重复触发验证。登录成功后，命令会读取网站写入的 `accessToken` 和 `refreshToken` Cookie、将凭证保存到本地，然后关闭窗口。该专用配置不会读取日常 Chrome 数据。

也可以保留旧版命令习惯，通过手机号和终端中的短信验证码登录：

```shell
ks login 13800138000
```

程序会在系统正常 Chrome 中打开一个本机临时页面，请先亲自完成腾讯人机验证。验证成功后程序自动发送短信，并在终端显示 `短信验证码发送成功，请输入 6 位验证码：`；输入手机收到的验证码并回车即可登录。这个流程不访问 `www.koushare.com`，因此不会触发网站首页的 EdgeOne 验证。

两种模式都不会破解、隐藏或代替人机验证。手机号模式中的本机回调仅监听 `127.0.0.1`，验证码结果只用于发送本次登录短信。

默认等待 10 分钟；可用 `--timeout 15m` 调整。手机号登录会打开一个没有地址栏和标签页的 Chrome 小窗口，滑块完成后自动关闭。如果没有自动找到 Chrome/Chromium，可用 `--browser-path /path/to/browser` 指定浏览器可执行文件。

无图形界面时，仍可通过环境变量导入已有的网页登录令牌：

```shell
KOUSHARE_ACCESS_TOKEN='...' KOUSHARE_REFRESH_TOKEN='...' ks login
```

`--access-token` / `--refresh-token` 和旧版 `--har` 仅作为兼容导入方式保留。命令行参数可能被 shell 历史记录，HAR 也包含敏感会话信息，因此日常使用优先选择浏览器登录。凭证文件权限会限制为当前用户可读写。

重复运行该命令会更新本地凭证。令牌过期后请再次运行 `ks login`。

### 1.2 注销登录状态

如果想注销登录状态，可以使用这条命令：

```shell
ks logout
```

手动删除程序所在路径下的`.ks.token`文件与该命令的执行效果相同。

## 二、查看视频或直播信息

**查看视频信息**使用`ks info [vid]`命令。

执行该命令后会返回标题、讲者、单位、日期、时长、地点、课件状态和视频简介。新版 API 的视频体积需要读取完整 HLS 清单，因此信息命令不再预先请求体积。


您可以试一试下面的例子：

```shell
ks info 7304
```

建议下载视频和课件前使用`info`命令确认视频的信息是否正确。

**查看直播信息**使用`ks info [liveID] --live`命令。执行该命令后会返回指定直播的标题、状态、主办方、开播时间、回放状态、观看次数、专题和最新通知。

您可以试一试下面的例子：

```shell
ks info 56428 --live
```

建议录制直播和下载快速回放前使用`info`命令确认直播的信息是否正确。

## 三、下载视频

**每个蔻享学术视频都有唯一对应的 id，即 vid。** 在蔻享学术网站进入某个视频的播放页面后，该页面网址的最后的数字部分即为该视频的 vid。例如，在下面的网址中，`7412`是该视频的 vid。

```
https://www.koushare.com/video/videodetail/7412
```

下载视频使用`ks save [vid] <flags>`命令。与`save`对应的主要 flag 如下：

| 简写形式 |   完整形式    |             说明              |   类型   |    默认值    |
| :------: | :-----------: | :---------------------------: | :------: | :----------: |
|   `-p`   |   `--path`    |      指定保存视频的路径       | `String` | 当前所在路径 |
|   `-q`   |  `--quality`  |     指定下载视频的清晰度      | `String` |     超清     |
|   `-s`   |  `--series`   |     指定是否下载专题视频      |  `Bool`  |      否      |
|   `-v`   | `--vidPrefix` | 指定是否使用vid作为文件名前缀 |  `Bool`  |      否      |
|   `-c`   | `--concurrency` | 单个视频同时下载的 HLS 片段数 |  `Int`  |      12      |
|    无    | `--video-concurrency` | 批量模式同时下载的视频数 |  `Int`  |      3       |

多个 flag 可以不分顺序地叠加使用，但`Bool`类型的 flag 宜放在最后使用。关于命令中 flag 的详细使用语法，可以参考[这里的描述](https://github.com/spf13/pflag#command-line-flag-syntax)。

### 3.1 使用默认参数下载视频

使用`save`时不加任何 flag ，程序就会使用`save`的所有 flag 的默认值进行下载。

例如，在登录状态下要默认下载 vid 为`7552`的视频，可以运行下面这条命令：

```shell
  ks save 7552
```

新版站点返回 HLS 媒体清单。该命令会并行下载并解密当前账户有权访问的片段，再严格按清单顺序合并为可直接播放的 `.ts` 文件。默认同时下载 12 个片段，可使用 `-c` 调整；源站限流时程序会自动换签并退避重试。

> `save`命令的别名是`video`，因此`ks save 7552`和`ks video 7552`的功能是相同的。

### 3.2 下载视频至指定文件夹

若要指定保存视频的位置，可以加上`-p`参数，并为其指定一个新值（如`D:\temp\`）以覆盖默认值（当前所在路径），如下所示：

```shell
  ks save 7552 -p D:\temp\
```

这里的`-p`是`--path`的简写形式，而`-p D:\temp\`与`--path=D:\temp\`是等价的，因此上一条命令也可以等价地修改为：

```shell
ks save 7552 --path=D:\temp\
```

若指定的文件夹不存在，程序会创建该文件夹以保存视频。若遇到`Access is denied`的错误提示，则说明权限不足，此时您需要使用更高的权限来运行 KouShare-dl。

### 3.3 下载某个专题的所有视频

专题下载需要指定`-s`参数，`-s`或`--series`参数是`Bool`型 flag，使用时无需指定具体的值。

您需要知道所要下载的专题视频中任意一个视频的 vid。以“中物院研究生院精品公开课之《高等量子力学》公开课程”专题为例，可以使用下面这条命令下载该专题的所有视频：

```shell
ks save 7304 -s
```

程序会使用该专题的名字创建一个文件夹用以存放下载的视频。`7304`是该专题第一个视频的 vid，可被替换为该专题任意视频的 vid。

若要同时指定保存视频的位置（如`D:\temp\`），可以运行该命令：

```shell
ks save 7552 -p D:\tmp\ -s
```

### 3.4 下载不同清晰度的视频

使用`-q`或`--quality`参数来指定下载视频的清晰度。该 flag 的值只有`high`（超清）、`standard`（高清）和`low`（标清）三种。示例如下：

```shell
ks save 7304 -q high
```

```shell
ks save 7304 -q standard
```

```shell
ks save 7304 --quality=low
```

需要注意的是：

- 若播放接口要求登录，请先导入网页端令牌；是否允许某个清晰度以服务器返回结果为准。
- 若参数不是 `high`、`standard` 或 `low`，程序会明确报错。
- 登录状态下，若您要下载的视频没有您指定的清晰度，程序会选择次于您指定清晰度的清晰度进行视频的下载。

### 3.5 批量下载指定的视频

`save` 命令的子命令 `batch` 可用于自定义批量下载指定 vid 的视频，格式如下：

```shell
ks save batch [vid1,vid2,vid3,...] <flag>
```

例如：

```shell
ks save batch [2233,59119,58206] -p="C:\Users\lenovo\Downloads" -v
```

KouShare-dl 默认同时下载 3 个视频，每个视频内部默认使用 12 路片段并发。例如，使用 4 个视频任务、每个视频 16 个片段并发：

```shell
ks save batch [2233,59119,58206] --video-concurrency 4 --concurrency 16
```

并发数过高可能触发源站限流；程序会自动换签、冷却并重试。可根据网络和源站情况降低这两个值。

## 四、录制直播与下载快速回放

**每个蔻享直播都有唯一的 liveID。** 当前网站直播详情页网址最后的数字就是 liveID。例如下方网址中的 `56428`：

```
https://www.koushare.com/live/details/56428
```

录制直播使用`ks record [liveID] <flags>`命令。

| 简写形式 |   完整形式    |                 说明                  |   类型   |    默认值    |
| :------: | :-----------: | :-----------------------------------: | :------: | :----------: |
|   `-@`   |    `--at`     | 开播时间，格式为"2006-01-02 15:04:05" | `String` | 立即开始录制 |
|   `-a`   | `--autoMerge` |  指定是否自动合并下载的视频片段文件   |  `Bool`  |      否      |
|   `-p`   |   `--path`    |        指定保存录制视频的路径         | `String` | 当前所在路径 |
|   `-r`   |  `--replay`   |    指定是否下载直播间快速回放视频     |  `Bool`  |      否      |
|          | `--password`  |            指定直播间密码             | `String` |              |

合并下载的`.ts`视频片段使用`ks merge <directory> <flags> `命令。与`merge`对应的 flag 有一个：

| 简写形式 | 完整形式 |                说明                |   类型   |          默认值          |
| :------: | :------: | :--------------------------------: | :------: | :----------------------: |
|   `-n`   | `--name` | 指定合并后文件的名字，格式`xxx.ts` | `String` | `recorded Video File.ts` |

### 4.1 对指定直播间进行录制

公开清晰度通常不需要登录；标为需登录的清晰度需要先导入令牌。例如：

```shell
  ks record 56428
```

执行命令后程序会立即开始录制。但如果此时尚未开播，您会收到“直播尚未开始”的提示，随后程序会自动倒计时至开播时间，倒计时结束后将自动开始录制。除了录制直播外，该命令也可用于查看回放视频是否上线等信息。

> `record`命令的别名是`live`。

如果直播尚未开始，但您知道准确的开播时间，那么可以用`-@`参数指定开播时间，如：

```shell
  ks record 56428 -@="2026-10-01 13:00:00"
```

运行后会倒计时，在指定时间重新检查直播状态，并选择当前登录状态可用的最高 HLS 清晰度。直播结束时自动停止录制并生成一个 `.ts` 文件。

> 注：若到指定的开播时间后直播间仍未开播，程序会自动退出。

若某个直播间需要密码才能访问，则需要使用 `--password` 标志指定正确的访问密码后才能对该直播间使用 `record` 命令。

### 4.2 合并录制的视频片段

新版录制器在下载过程中直接把 HLS 片段写入一个 `.ts` 文件，因此 `--autoMerge` 仅为兼容旧脚本而保留。`merge` 命令仍可用于合并旧版本留下的多个 `.ts` 文件。

有时直播时间过长，自动合并后得到的文件体积较大，不便于传输，可以在录制直播时不指定`-a`参数，这样下载下来的直播片段不会自动合并。您可以在传输后使用`merge`命令手动合并`.ts`视频片段：

```shell
  ks merge <directory> <flags>
```

其中`<directory>`参数为存放视频片段文件的文件夹的路径，若为空则默认为程序当前所在路径。

示例如下：

```shell
ks merge
```

```shell
ks merge D:\temp\直播录制 -n 课程.ts
```

```shell
ks merge -n output.ts
```

### 4.3 下载直播间快速回放视频

当直播详情 API 已返回标准 HLS 回放地址，或已关联正式视频 ID 时，可使用：

```shell
ks record 56428 --replay
```

可使用 `-p` 指定保存路径。新版快速回放列表可以公开读取，但取得实际播放地址要求登录；请先用 `ks login` 导入您自己的网页登录凭证。若一场直播有多个快速回放片段，程序会逐个下载为独立的 `.ts` 文件。

```shell
ks record 56428 --replay -p "C:\Users\lenovo\Desktop"
```

## 五、下载课件

下载课件使用`ks slide [vid] <flags>`命令。与`slide`对应的 flag 有三个：

| 简写形式 |   完整形式   |              说明              |   类型   |    默认值    |
| :------: | :----------: | :----------------------------: | :------: | :----------: |
|   `-p`   |   `--path`   |       指定保存课件的路径       | `String` | 当前所在路径 |
|    无    | `--qpdf-bin` | 指定qpdf的bin文件夹所在的路径  | `String` |  不使用qpdf  |
|   `-s`   |  `--series`  | 指定是否下载整个专题的所有课件 |  `Bool`  |      否      |

### 5.1 下载单个课件和专题课件

下载课件时不需要处于登录状态下。假如您想要下载为 vid 为`7405`的视频关联的课件，可以运行该命令：

```shell
ks slide 7405
```

使用`info`命令查看 vid 为`7405`的视频信息，可以发现该视频的“专题”不为空，说明该视频还有其它相关视频。

假如您想下载这个专题视频的所有课件，可以使用`-s`参数：

```shell
ks slide 7405 -s
```

同样地，`7405`可以被替换为同专题任意视频的 vid。

### 5.2 优化 pdf 文件【实验性功能】

该功能当前并不稳定，不推荐使用。

如果想要使用`--qpdf-bin`标志，需先下载 [qpdf包](https://github.com/qpdf/qpdf/releases/latest) 并进行解压操作，然后在命令行或终端中指定 qpdf 包的 bin 文件夹所在的路径，如：

```shell
ks slide 7405 --qpdf-bin=C:\Downloads\qpdf-10.1.0\bin\
```

# 六、清理临时文件

使用 `ks clean` 命令可以清理当前目录或指定路径下的所有下载过程中产生的 `tmp` 文件。与 `clean` 对应的 flag 有两个：

| 简写形式 | 完整形式  |              说明              |   类型   |    默认值    |
| :------: | :-------: | :----------------------------: | :------: | :----------: |
|   `-p`   | `--path`  |     指定清理临时文件的路径     | `String` | 当前所在路径 |
|   `-q`   | `--quiet` | 指定是否不输出清理过程中的信息 |  `Bool`  |      否      |

# FAQ

#### KouShare-dl 下载视频时是并行下载吗？
是。单视频默认以 12 路并发下载 HLS 片段，并在内存中排序后依次写入；`save batch` 默认同时下载 3 个视频。可分别使用 `--concurrency` 和 `--video-concurrency` 调整。

#### 下载专题视频时因网络波动导致下载中断该怎么办？
再次运行命令会跳过已经完整下载并完成重命名的视频。新版 HLS 的单个视频若仅留下 `.tmp` 文件，会从头重新下载，以避免重复或缺失加密片段。

#### 下载视频的过程中遇到因被占用而导致文件重命名失败的错误应该如何处理？
错误信息通常为：`rename 文件名.ts.tmp 文件名.ts: The process cannot access the file because it is being used by another process.`请关闭占用该文件的播放器或同步程序后，手动去掉 `.tmp` 后缀；也可以重新运行下载命令。

# 鸣谢

特别感谢 [JetBrains](https://www.jetbrains.com/) 提供的 [GoLand](https://www.jetbrains.com/go) 等 IDE 的授权。
特别感谢为 KouShare-dl 预览版本测试各项功能的小伙伴们。

# 许可证合规性

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fyliu7949%2FKouShare-dl.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fyliu7949%2FKouShare-dl?ref=badge_large)
