## Qiniu-cert

通过let's encrypt免费自动生成https证书，并上传到七牛（qiniu.com)，绑定存储空间的cdn加速域名。

本程序主要是方便 run,renew 的自动执行。

### 前置条件
1. 拥有自己的域名
2. 在七牛网站手动设置好域名
3. 在本机安装[lego](https://github.com/xenolf/lego)【注：就是一个可执行文件而已】

### 编译生成二进制
```
go build -o ~/Desktop/qcert *.go
```
生成的目录按需修改

### 运行
```
ALICLOUD_ACCESS_KEY=xxxxx  ALICLOUD_SECRET_KEY=yyyyy ./qcert --email="your-email" --domains="*.cdn.yourdomain.com" --dns="alidns" --certname="cert1" --qiniudomain=".cdn.yourdomain.com"
```

例子里，我用的是阿里云的域名管理，按照其官网要求可以获取`ALICLOUD_ACCESS_KEY`和`ALICLOUD_SECRET_KEY`。

* `--domains`是证书本身认证的域名
* `--renew`在延长证书有效期的时候使用
* `--path`表示证书在本地存储位置
* `--skipnewcert`表示最新的证书本地已经存在，不用再生成新的了。一般不用设置。
* `--dns`指定域名服务商的提供者，与`lego`的设置相同
* `--certname`是七牛上用到的名字，随便取一个就行
* `--qiniudomain`是七牛上指定的域名，一般与`domains`相同。但`domains`是泛域名，`qiniudomain`是某个域名时，它们不同。例如，`qiniudomain`是`a.xxx.yourdomain.com`，`domains`是`*.xxx.yourdomain.com`

### 执行过程
其实这个工具只是对 证书认证，证书上传，绑定证书到七牛域名这几件事做了封装，方便后期自动化。
相关部分技术说明见[博客](https://www.jianshu.com/p/8e91e4e7e703)。

#### 证书认证
就是调用`lego`，需要域名服务商的api作自动化。相关参数`--email`，`--domains`，`--dns`，`--renew`，`--path`，`--skipnewcert`。 

#### 上传证书
调用七牛api。相关参数`--certname`，`--existcertid`。

#### 绑定证书
调用七牛api。相关参数`--certname`，`--qiniudomain`。