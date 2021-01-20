# KouShare-dl
[![License](https://img.shields.io/github/license/yliu7949/KouShare-dl.svg)](https://github.com/yliu7949/KouShare-dl/blob/master/LICENSE)

KouShare-dl是一个用于从[“蔻享学术”](https://www.koushare.com/)视频网站下载视频的CLI工具。

您可以在常见的桌面操作系统（Windows，macOS和Linux的各个发行版）里使用该命令行工具。该工具已被发布至公共领域，因此您可以按照您的想法自由使用它，如对它进行修改、重新发布等操作。

# 功能
### 它目前具有如下功能：
- 获取视频的详细信息
- 下载单个视频
- 下载整个系列的视频
- 继续上一次的下载

### 它**无法**做到的事情：
- 下载付费视频
- 下载清晰度高于标清的视频

# 编译
如果您是Windows平台用户，可以直接下载Releases中的可执行文件。否则，您需要下载源代码自行编译。

# 用法
您需要通过命令行或终端进入该工具所在的文件夹，才能执行相关命令。

命令格式：
```
  ks [command] <flag>
```
可使用的command：
```
  help        查看某个具体命令的更多帮助信息
  info        获取视频的基本信息
  save        保存指定vid的视频（vid为视频网址里最后面的一串数字）
```
可使用的flag：
```
  -h, --help      查看帮助信息
  -p, --path      指定保存视频的路径（若不指定，则默认为该程序当前所在的路径）
  -s, --series    指定是否下载整个系列的视频
  -v, --version   查看版本号
```

# 示例
### 1、下载vid为7552的视频
```
  ks save 7552
```
若要指定保存视频的位置（如`D:\tmp\`），可以加上`-p`参数
```
  ks save 7552 -p D:\tmp\
```
### 2、下载vid为7552的视频所在系列的所有视频
```
  ks save 7552 -s
```
若要同时指定保存视频的位置（如`D:\tmp\`），可以使用
```
  ks save 7552 -p D:\tmp\ -s
```
### 3、查看vid为7552的视频的详细信息
```
  ks info 7552
```
`info`命令可以输出视频的讲者、报告日期、所在系列名、视频简介以及视频时长、体积等信息。
### 4、查看某个命令（如`save`命令）的详细帮助信息
```
  ks help save
```
或者使用`ks save --help`。
### 5、查看该工具的版本信息
```
  ks -v
```

# FAQ
#### KouShare-dl下载视频时是并行下载吗？
不是并行下载。
#### 下载系列视频时因网络波动导致下载中断该怎么办？
再次运行您上一次使用的下载命令，KouShare-dl会自动跳过已下载完成的视频，并继续完成您的下载。

# 鸣谢
特别感谢 [JetBrains](https://www.jetbrains.com/) 提供的 [GoLand](https://www.jetbrains.com/go) 等 IDE 的授权。
特别感谢为KouShare-dl v0.1版本测试各项功能的小伙伴们。
