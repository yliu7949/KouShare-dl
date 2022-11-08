<p align="center">
  <img width="240" src="./logo.png" style="text-align: center;" alt="KouShare-dl logo">
</p>

# KouShare-dl

[![License](https://img.shields.io/github/license/yliu7949/KouShare-dl.svg)](https://github.com/yliu7949/KouShare-dl/blob/master/LICENSE)
[![Build Status](https://github.com/yliu7949/KouShare-dl/workflows/Go/badge.svg)](https://github.com/yliu7949/KouShare-dl/actions?query=workflow%3AGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/yliu7949/KouShare-dl)](https://goreportcard.com/report/github.com/yliu7949/KouShare-dl)
[![Github Downloads](https://img.shields.io/github/downloads/yliu7949/KouShare-dl/total.svg)](http://gra.caldis.me/?url=https://github.com/yliu7949/KouShare-dl)
<a title="Hits" target="_blank" href="https://github.com/yliu7949/KouShare-dl"><img src="https://hits.b3log.org/yliu7949/KouShare-dl.svg"></a>
[![Github Release Version](https://img.shields.io/github/v/release/yliu7949/KouShare-dl?color=green&include_prereleases)](https://github.com/yliu7949/KouShare-dl/releases/latest)


KouShare-dl 是一个使用 [Cobra](https://github.com/spf13/cobra)
开发的用于从 [“蔻享学术”](https://www.koushare.com/) 视频网站下载视频和课件的 CLI 工具。

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
  * [四、录制直播与下载快速回放](#四录制直播与下载快速回放)
    + [4.1 对指定直播间进行录制](#41-对指定直播间进行录制)
    + [4.2 合并录制的视频片段](#42-合并录制的视频片段)
    + [4.3 下载直播间快速回放视频](#43-下载直播间快速回放视频)
  * [五、下载课件](#五下载课件)
    + [5.1 下载单个课件和专题课件](#51-下载单个课件和专题课件)
    + [5.2 优化 pdf 文件【实验性功能】](#52-优化-pdf-文件实验性功能)
- [FAQ](#faq)
    - [KouShare-dl 下载视频时是并行下载吗？](#koushare-dl-下载视频时是并行下载吗)
    - [下载专题视频时因网络波动导致下载中断该怎么办？](#下载专题视频时因网络波动导致下载中断该怎么办)
    - [下载视频的过程中遇到因被占用而导致文件重命名失败的错误应该如何处理？](#下载视频的过程中遇到因被占用而导致文件重命名失败的错误应该如何处理)
- [鸣谢](#鸣谢)
- [许可证合规性](#许可证合规性)

# 功能

### 它目前具有如下功能

- 登录蔻享账户，且一周内免登录

- 获取视频或直播的详细信息

- 下载单个蔻享视频或整个专题的视频

- 下载清晰度为标清、高清和超清的视频（需要登录）

- 下载**已购买且在有效期内**的付费视频（需要登录）

- 继续上一次的视频下载

- 定时录制直播间

- 继续上一次的直播间录制

- 下载直播间的快速回放🚀

- 下载单个课件或整个专题的课件

### 它无法做到的事情

- 下载未购买的付费视频

### "功能支持"表格

| 类型 | 是否支持专题下载 | 是否支持单独下载 | 是否支持断点续传 | 是否支持不同清晰度的下载 | 是否支持付费产品下载 |
| :--: | :--------------: | :--------------: | :--------------: | :----------------------: | :------------------: |
| 视频 |        ✔️         |        ✔️         |        ✔️         |            ✔️             |          ⭕           |
| 直播 |        ➖         |        ✔️         |        ✔️         |            ❌             |          ➖           |
| 课件 |        ✔️         |        ✔️         |        ❌         |            ➖             |          ✔️           |

（✔️表示支持该功能，❌表示不支持该功能，➖表示该功能不存在，⭕表示部分支持该功能）

# 编译

您可以下载 [Releases](https://github.com/yliu7949/KouShare-dl/releases/latest)
中的二进制文件`ks.exe`或`ks`后直接使用，也可以下载源代码自行编译。
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

若出现`KouShare-dl v0.9.0`字样，则说明可以正常使用。接下来您可以继续输入 KouShare-dl 程序的命令来进行交互。比如，输入`ks help`并回车，您就可以看到 KouShare-dl 程序的帮助信息了。

# 命令简介

KouShare-dl 程序的命令具有下面的格式：

```shell
  ks [command] <flag>
```

其中`[command]`为必选命令，`<flag>`为可选参数。

可使用的 command 命令：

```shell
  help        查看某个具体命令的更多帮助信息
  info        获取视频或直播的基本信息
  login       通过短信验证码获取“蔻享学术”登陆凭证
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
  -h, --help        查看帮助信息
  -n, --name        指定输出文件的名字
  -p, --path        指定保存文件的路径（若不指定，则默认为该程序当前所在的路径）
  -P, --proxy       指定使用的http/https/socks5代理服务地址
  -q, --quality     指定下载视频的清晰度（high为超清，standard为高清，low为标清，不指定则默认为超清）
      --qpdf-bin    指定qpdf的bin文件夹所在的路径（注：该flag无简写形式）
  -r, --replay      指定是否下载直播间快速回放视频
  -s, --series      指定是否下载整个专题的文件
      --nocolor     指定是否不使用彩色输出
  -v, --version     查看版本号
```

需要注意的是，对于每个 command 命令，仅有部分 flag 参数是可用且有效的。可以通过`ks help [command]`来查看某个命令的详细描述及其可用的 flag 参数。

# 示例

## 一、登录账户与注销登陆

登录蔻享账户并不是使用流程中的必须操作，但登录蔻享账户后可以下载更高清晰度的视频和下载已购买的付费视频，获取视频的基本信息时还可以获取到更多详细的内容。

### 1.1 登录蔻享账户

使用下面的命令登录蔻享账户：

```shell
ks login [phone number]
```

其中`[phone number]`参数为 11 位手机号码。该命令执行后，手机会收到 6 位短信验证码，在命令行中继续输入短信验证码后回车即可登录。登录成功后会在当前路径下生成一个用于保存登录凭证的 Token 隐藏文件，Token 有效期为一周，因此一周内无需再次登录即可保持登录状态。

重复运行该命令会自动更新登陆凭证。登录凭证过期后重新登陆即可。

### 1.2 注销登录状态

如果想注销登录状态，可以使用这条命令：

```shell
ks logout
```

手动删除程序所在路径下的`.token`文件与该命令的执行效果相同。

## 二、查看视频或直播信息

**查看视频信息**使用`ks info [vid]`命令。`info`命令没有 flag 。

执行该命令后会返回指定 vid 的视频的详细信息，包括标题、讲者、单位、日期、时长、体积、类别、专题、分组以及视频简介等。

几点说明：

- 非登录状态下，“体积”仅展示标清清晰度下的视频大小；登录状态下，“体积”展示最高清晰度下的视频大小。

- 若“体积”为`0MB [未知]`，则说明该视频是未购买的（或未在购买有效期内的）付费视频，此时 KouShare-dl 无法获取该视频的体积信息。

- 若“专题”不为空，说明该视频是属于某个专题的视频，比如某精品公开课中的一节课。


您可以试一试下面的例子：

```shell
ks info 7304
```

建议下载视频和课件前使用`info`命令确认视频的信息是否正确。

**查看直播信息**使用`ks info [roomID]`命令。执行该命令后会返回指定 roomID 的直播间的详细信息，包括标题、直播状态、主办方、开播时间、有无回放、浏览次数、专题以及最新通知等。

您可以试一试下面的例子：

```shell
ks info 341215
```

建议录制直播和下载快速回放前使用`info`命令确认直播的信息是否正确。

## 三、下载视频

**每个蔻享学术视频都有唯一对应的 id，即 vid。** 在蔻享学术网站进入某个视频的播放页面后，该页面网址的最后的数字部分即为该视频的 vid。例如，在下面的网址中，`7412`是该视频的 vid。

```
https://www.koushare.com/video/videodetail/7412
```

下载视频使用`ks save [vid] <flags>`命令。与`save`对应的 flag 有三个：

| 简写形式 |  完整形式   |         说明         |   类型   |    默认值    |
| :------: | :---------: | :------------------: | :------: | :----------: |
|   `-p`   |  `--path`   |  指定保存视频的路径  | `String` | 当前所在路径 |
|   `-q`   | `--quality` | 指定下载视频的清晰度 | `String` |     超清     |
|   `-s`   | `--series`  | 指定是否下载专题视频 |  `Bool`  |      否      |

多个 flag 可以不分顺序地叠加使用，但`Bool`类型的 flag 宜放在最后使用。关于命令中 flag 的详细使用语法，可以参考[这里的描述](https://github.com/spf13/pflag#command-line-flag-syntax)。

### 3.1 使用默认参数下载视频

使用`save`时不加任何 flag ，程序就会使用`save`的所有 flag 的默认值进行下载。

例如，在登录状态下要默认下载 vid 为`7552`的视频，可以运行下面这条命令：

```shell
  ks save 7552
```

该命令执行完毕后，程序所在的路径下会出现一个`.mp4`格式的超清视频文件，这就是下载下来的 vid 为`7552`的蔻享视频。

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

- 非登录状态下，`-q`和`--quality`参数无效。这是因为非登录状态下仅能下载标清视频。
- 若您指定的该 flag 的值并不在以上三种值之内，程序会判定要下载的清晰度为标清。
- 登录状态下，若您要下载的视频没有您指定的清晰度，程序会选择次于您指定清晰度的清晰度进行视频的下载。

## 四、录制直播与下载快速回放

**每个蔻享直播间都有唯一对应的 id，即 roomID。** 在蔻享学术网站进入某个直播间的页面后，该页面网址的最后的数字部分即为该直播间的房间号。例如，在下面的网址中，`676216`是该直播间的 roomID。

```
https://www.koushare.com/lives/room/676216
```

录制直播使用`ks record [roomID] <flags>`命令。与`record`对应的 flag 有三个：

| 简写形式 |   完整形式    |                 说明                  |   类型   |    默认值    |
| :------: | :-----------: | :-----------------------------------: | :------: | :----------: |
|   `-@`   |    `--at`     | 开播时间，格式为"2006-01-02 15:04:05" | `String` | 立即开始录制 |
|   `-a`   | `--autoMerge` |  指定是否自动合并下载的视频片段文件   |  `Bool`  |      否      |
|   `-p`   |   `--path`    |        指定保存录制视频的路径         | `String` | 当前所在路径 |
|   `-r`   |  `--replay`   |    指定是否下载直播间快速回放视频     |  `Bool`  |      否      |

合并下载的`.ts`视频片段使用`ks merge <directory> <flags> `命令。与`merge`对应的 flag 有一个：

| 简写形式 | 完整形式 |                说明                |   类型   |          默认值          |
| :------: | :------: | :--------------------------------: | :------: | :----------------------: |
|   `-n`   | `--name` | 指定合并后文件的名字，格式`xxx.ts` | `String` | `recorded Video File.ts` |

### 4.1 对指定直播间进行录制

录制直播时不需要处于登录状态下。假如您想要录制房间号为`751111`的直播间，可以运行该命令：

```shell
  ks record 751111 -a
```

执行命令后程序会立即开始录制。但如果此时尚未开播，您会收到相关提示（距离开播还有一段时间、正式回放视频已上线等），随后程序会自动退出。该命令用于录制已开始直播的直播间，或者查看回放视频是否上线等信息。

> `record`命令的别名是`live`，所以` ks record 751111 -a`和` ks live 751111 -a`的功能是相同的。

如果直播尚未开始，但您知道准确的开播时间，那么可以用`-@`参数指定开播时间，如：

```shell
  ks record 751111 -@="2021-07-15 18:30:00" -a
```

运行这条命令后会立即启动倒计时，到指定的开播时间后 KouShare-dl 会以`1080p`的清晰度自动开始录制直播，直播结束时会自动停止录制。

> 注：若到指定的开播时间后直播间仍未开播，程序会自动退出。

### 4.2 合并录制的视频片段

在观看直播时，直播视频是以一个个小文件（即一些时长较短的视频片段）的方式传输给用户的。在上一个示例中，指定`-a`参数后，KouShare-dl 会自动合并下载的直播视频片段为一个`.ts`文件（一种视频文件，可被视频播放器直接播放）。

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

**示例：** 使用`ks live 447482 `命令得到“快速回放视频已上线”的信息：

```bash
$ ks live 447482

直播已结束。快速回放视频已上线，访问 https://www.koushare.com/lives/room/447482 观看快速回放或使用“ks record 447482 --replay”命令下载快速回放视频。
```

使用`ks live 447482 -r `或`ks record 447482 --replay`命令即可下载快速回放视频：

```bash
$ ks live 447482 -r

开始下载快速回放视频...
2126692489_2083434824_1.ts?start=0 ...
2126692489_2083434824_1.ts?start=1752160 ...
2126692489_2083434824_1.ts?start=3504696 ...
 ...
快速回放视频下载完成。
```

可使用`-p`指定保存快速回放视频的路径，如：

```bash
ks live 447482 -r -p "C:\Users\lenovo\Desktop"
```

```bash
ks live 447482 -r --path="C:\Users\lenovo\Desktop"
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

# FAQ

#### KouShare-dl 下载视频时是并行下载吗？
不是并行下载。

#### 下载专题视频时因网络波动导致下载中断该怎么办？
再次运行您上一次使用的下载命令，KouShare-dl 会自动跳过已下载完成的视频，并继续完成您的下载。
录制直播意外中断时同理。

#### 下载视频的过程中遇到因被占用而导致文件重命名失败的错误应该如何处理？
错误信息通常为：`rename 文件名.tmp 文件名.mp4: The process cannot access the file because it is being used by another process.`您可以耐心等待至下载结束后，手动将重命名失败的`.tmp`文件的后缀改为`.mp4`；或者重新运行您上一次使用的下载命令，KouShare-dl 会再次尝试重命名这些`.tmp`文件。

# 鸣谢

特别感谢 [JetBrains](https://www.jetbrains.com/) 提供的 [GoLand](https://www.jetbrains.com/go) 等 IDE 的授权。
特别感谢为 KouShare-dl 预览版本测试各项功能的小伙伴们。

# 许可证合规性

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fyliu7949%2FKouShare-dl.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fyliu7949%2FKouShare-dl?ref=badge_large)
