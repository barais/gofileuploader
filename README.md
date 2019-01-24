# Project overview

This project provides an infrastructure for rendering and automatic evaluation of lab work in computer science. 

This project is developed using GoLang.

It consists of a [library](https://github.com/barais/ipfilter/) to open the web application based on a set of IP addresses and time range, a [module](https://github.com/barais/gofileuploader/) to provide a web application to allow project loading and a [module](https://github.com/barais/gomavenworker/) to evaluate student projects.

sIt uses rabbitmq for communication between the different modules.  

The architecture is the following:

![ArchiRenduTP](doc/img/ArchiRenduTP.png)

# Architecture overview

## GoFileUploader

The main features of the GoFileUploader module are the following:

- Filter By IP And Date(Open app only for specific lab room and lab schedule)
- Protection behind CAS
- LDAP Connexion To get students Info Based on Cas login
- Custom Html Page
- Upload File
- Check File (valid zip, min size, max size)
- Send Email to Student
- Provide Immediate Feedback

## GoFileWorker

The main features of the GoFileWorker module are the following:

- Uncompress file
- Create valid maven project based on template
- Build project using maven
- Check style using Maven
- Run tests (come from maven template) using Maven
- Analyze tests and checkstyle results 
      (query output file using XQuery)
- Compute Grade
- Send an Email with result to students
- Save outputs in CSV

# Installation

##  GoFileUploader

```bash
git get github.com/barais/gofileuploader
cd $GOPATH/src/github.com/barais/gofileuploader
go build
./gofileuploader
```

**Typical usage**

```bash
./gofileuploadernew -p 8094 -queue rabbitmqNameOfTheQueue -login "YOURSMTPLOGIN"  -pass "YOURSMTPPATH"  -ipfilterconfig ipfilter.json -d public/ -u upload/

```

**Parameter**

```txt
Usage of ./gofileuploader:
  -alsologtostderr
    	log to standard error as well as files
  -amqp string
    	amqp server to use (default "amqp://localhost:5672/")
  -buildproject
    	Build project (default true)
  -d string
    	the directory of static file to host (default "./public")
  -ipfilterconfig string
    	json file to configure ipfilter (default "ipfilter.json")
  -ldapserver string
    	ldap server to use (default "ldap.univ-rennes1.fr:389")
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory
  -login string
    	login of smtp server (default "obarais")
  -logtostderr
    	log to standard error instead of files
  -p string
    	port to serve on (default "8080")
  -pass string
    	pass of smtp server
  -queue string
    	queue name to use (default "si2")
  -sendemail
    	Send an email (default true)
  -smtpserver string
    	smtp server to use (default "smtps.univ-rennes1.fr:587")
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -u string
    	path of the folder to upload file (default "upload")
  -url string
    	the URL of your cas server (default "https://sso-cas.univ-rennes1.fr")
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging

```


## GoFileWorker


```bash
git get github.com/barais/gofileworker
cd $GOPATH/src/github.com/barais/gofileworker
go build
./gofileworker
```

**Typical usage**

```bash
./gofileworker -queue rabbitmqNameOfTheQueue -login "YOURSMTPLOGIN"  -pass "YOURSMTPPATH"  -maven /opt/maven/bin/ -templatePath templateProjectDemo/

```

**Parameter**

```txt
Usage of ./gofileworker:
  -amqp string
    	amqp server to use (default "amqp://guest:guest@localhost:5672/")
  -corrige
    	provide a grade
  -corrigeCP string
    	provide a CP name
  -login string
    	login of smtp server (default "obarais")
  -maven string
    	path to maven executable (default "/opt/apache-maven-3.5.0/bin")
  -outputfile string
    	path of the csv file to put results (default "/tmp/result.csv")
  -pass string
    	pass of smtp server
  -queue string
    	queue name to use (default "si2")
  -sendemail
    	Send an email (default true)
  -smtpserver string
    	smtp server to use (default "smtps.univ-rennes1.fr:587")
  -templatePath string
    	path to the template project that contains tests and lib (default "templateProject")
```

## Supervisor

```bash
sudo apt-get install supervisor
```

Edit config file

```bash
nano -w /etc/supervisor/conf.d/gofileuploader.conf
```

Sample of supervisor (folder name must be adapted to your installation)

```txt
[program:gofileuploaderTest]
directory=/opt/gofileuploaderapp
command=/opt/gofileuploaderapp/gofileuploader -p 9501 -queue test -login "YOURSMTPLOGIN"  -pass "YOURSMTPPATH" -ipfilterconfig /opt/gofileuploaderapp/ipfilterTest.json -d /opt/gofileuploaderapp/publicDemo/ -u /opt/gofileuploaderapp/upload/Demo
autostart=true
autorestart=true
redirect_stderr=true

[program:gofileuploaderTestmavenbuild]
directory=/opt/gofileuploaderapp
command=/opt/gofileuploaderapp/gofileworker -queue test -login "YOURSMTPLOGIN"  -pass "YOURSMTPPATH"   -maven /opt/maven/bin/ -templatePath /opt/gofileuploaderapp/templateProjectDemo
autostart=true
autorestart=true
redirect_stderr=true
```


## RabbitMQ Server

```bash
sudo apt-get install rabbitmq-server
```

 /etc/rabbitmq/rabbitmq-env.conf

```txt
# Defaults to rabbit. This can be useful if you want to run more than one node
# per machine - RABBITMQ_NODENAME should be unique per erlang-node-and-machine
# combination. See the clustering on a single machine guide for details:
# http://www.rabbitmq.com/clustering.html#single-machine
#NODENAME=rabbit

# By default RabbitMQ will bind to all interfaces, on IPv4 and IPv6 if
# available. Set this if you only want to bind to one network interface or#
# address family.
NODE_IP_ADDRESS=127.0.0.1

# Defaults to 5672.
NODE_PORT=5672

```

```bash
sudo service rabbitmq-server restart
```

## nginx

sample of nginx proxy. Change APP.yourappdomainname.fr with your app domain. Use letsencrypt if you need https. 


```bash
server {
        listen   80;
        server_name     APP.yourappdomainname.fr;
	 location / {
		proxy_connect_timeout 159s;
                proxy_send_timeout   600;
                proxy_read_timeout   600;
                proxy_http_version 1.1;
                proxy_pass         http://127.0.0.1:9501/;
		proxy_set_header Host APP.yourappdomainname.fr;
    	proxy_set_header X-Host APP.yourappdomainname.fr;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
#		proxy_set_header X-Forwarded-Proto $proxy_x_forwarded_proto;
		proxy_buffering off;
        }
    error_page 502 /HTTP502.html;
    location /HTTP502.html {
        root /etc/nginx/errorpages/HttpErrorPages/dist/;
    }

}
```


An example of HTTP502 is available [here](https://github.com/barais/gofileuploader/tree/master/nginxerrorpage) 