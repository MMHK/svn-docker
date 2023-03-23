# SVN Docker

Subverison service with docker

[![dockeri.co](https://dockeri.co/image/mmhk/svn-docker)](https://hub.docker.com/r/mmhk/svn-docker)

![GitHub](https://img.shields.io/github/license/mmhk/svn-docker)


## How to use


1. run docker 
```
docker run -d -v [your svn root]:/var/svndata -p 80:80 mmhk/svn-docker
```
2. open http://localhost/svnadmin, SVN admin web Panel
3. svn client open repository url as http://localhost/svn/[your repository]


###  Environment

|ENV|default|remark|
|-|-|-|
|WWW_UID|1000|apache user ID 当你需要匹配现有svn仓库 UID的时候|
|WWW_GID|1000|apache group ID 当你需要匹配现有svn仓库 UID的时候|
|SVN_USER|mixmedia|初始仓库管理员账户|
|SVN_PWD|123456|初始仓库管理员密码|
