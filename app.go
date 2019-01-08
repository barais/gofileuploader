package main

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	// Error handling
	"errors"

	"github.com/thanhpk/randstr"

	// To read password from standard input without echoing
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	//"html/template"
	"net/url"

	"github.com/adelowo/filer"
	"github.com/adelowo/filer/validator"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/barais/ipfilter"
	"github.com/otiai10/copy"
	"gopkg.in/cas.v2"
	"gopkg.in/ldap.v3"
)

//TODO
// manage maven crash

var fileserver http.Handler
var login string
var pass string
var uploadfolder string
var mavenhome string
var templateProjectPath string
var smtpserver string
var ldapserver string
var sendemail bool
var buildproject bool
var ipfilterconfig string

func main() {
	port := flag.String("p", "8080", "port to serve on")
	directory := flag.String("d", "./public", "the directory of static file to host")
	//dir = *directory
	casURL := flag.String("url", "https://sso-cas.univ-rennes1.fr", "the URL of your cas server")
	paramlogin := flag.String("login", "obarais", "login of smtp server")
	parampass := flag.String("pass", "", "pass of smtp server")
	uploadfolderparam := flag.String("u", "upload", "path of the folder to upload file")
	mavenhomeparam := flag.String("maven", "/opt/apache-maven-3.5.0/bin", "path to maven executable")
	templateProjectPathparam := flag.String("templatePath", "templateProject", "path to the template project that contains tests and lib")
	smtpserverparam := flag.String("smtpserver", "smtps.univ-rennes1.fr:587", "smtp server to use")
	ldapserverparam := flag.String("ldapserver", "ldap.univ-rennes1.fr:389", "ldap server to use")
	sendEmailparam := flag.Bool("sendemail", true, "Send an email")
	buildProjectparam := flag.Bool("buildproject", true, "Build project")
	ipfilterconfigparam := flag.String("ipfilterconfig", "ipfilter.json", "json file to configure ipfilter")
	flag.Parse()
	fileserver = http.FileServer(http.Dir(*directory))
	login = *paramlogin

	uploadfolder = *uploadfolderparam
	mavenhome = *mavenhomeparam
	templateProjectPath = *templateProjectPathparam
	smtpserver = *smtpserverparam
	ldapserver = *ldapserverparam
	sendemail = *sendEmailparam

	// Password initialisation for smtp
	pass = *parampass
	if sendemail {
		if pass == "" { // Reading password from standard input
			fmt.Println("Enter your smtp password: ")
			password, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				fmt.Println("Error while reading your smtp password")
				fmt.Println(err)
			}
			pass = string(password)
			if pass == "" {
				fmt.Println("Error while reading your smtp password. Empty password.")
			}
		}
	}

	buildproject = *buildProjectparam
	ipfilterconfig = *ipfilterconfigparam
	mux := http.NewServeMux()

	ipfilterconfigfile, err := os.Open(ipfilterconfig)
	if err != nil {
		log.Fatal(err)
	}
	defer ipfilterconfigfile.Close()
	allowedSchedule, err := ioutil.ReadAll(ipfilterconfigfile)
	var allowedScheduleO []*ipfilter.IPInterval
	json.Unmarshal([]byte(allowedSchedule), &allowedScheduleO)

	mux.HandleFunc("/", testCas)
	url, _ := url.Parse(*casURL)
	client := cas.NewClient(&cas.Options{
		URL: url,
	})

	options := ipfilter.NewOption(allowedScheduleO)

	myProtectedHandler := ipfilter.Wrap(client.Handle(mux), *options)

	log.Printf("Serving %s on HTTP port: %s\n", *directory, *port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+*port, myProtectedHandler))
}

type templateBinding struct {
	Username   string
	Attributes cas.UserAttributes
}

func testCas(w http.ResponseWriter, r *http.Request) {
	if !cas.IsAuthenticated(r) {
		cas.RedirectToLogin(w, r)
		return
	}
	if r.URL.Path == "/logout" {
		cas.RedirectToLogout(w, r)
		return
	}
	log.Println(cas.Username(r))
	binding := &templateBinding{
		Username:   cas.Username(r),
		Attributes: cas.Attributes(r),
	}
	if r.URL.Path == "/upload" {
		uploadProgress(w, r, binding)
		return
	}

	fileserver.ServeHTTP(w, r)
}

func uploadProgress(w http.ResponseWriter, r *http.Request, binding *templateBinding) {
	mr, err := r.MultipartReader()
	resultNumber := 0
	if err != nil {
		resultNumber = -1
		return
	}
	//    length := r.ContentLength
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		token := randstr.String(24) // generate a random 16 character length string
		timestamp := time.Now().Format("20060102150405")
		var read int64
		//		var p float32
		path := uploadfolder + "/" + binding.Username + "_" + timestamp + "_" + token + ".zip"
		dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("Unable to create file " + path)
			log.Printf("Local directory " + uploadfolder + " probably unexisting.")
			log.Printf("Please create one, and re-start server with option -u set to the created path.")
			resultNumber = -1
			return
		}
		for {
			buffer := make([]byte, 1000000)
			cBytes, err := part.Read(buffer)
			read = read + int64(cBytes)
			//fmt.Printf("read: %v \n",read )
			//p = float32(read) / float32(length) *100
			//fmt.Printf("progress: %v \n",p )
			dst.Write(buffer[0:cBytes])
			if err == io.EOF {
				break
			}
		}
		dst.Close()

		max, _ := filer.LengthInBytes("20MB")
		min, _ := filer.LengthInBytes("1KB")
		var val2 = validator.NewSizeValidator(max, min)
		var file, _ = os.Open(path)
		var val1 = validator.NewMimeTypeValidator([]string{"application/zip"})
		var val3 = validator.NewExtensionValidator([]string{"zip"})
		//var val2 = validator.NewSizeValidator((1024 * 1024 * 10), (1 * 1)) //10MB(maxSize) and 1B(minSize)

		var errg error
		if _, err := val1.Validate(file); err != nil {
			errg = err
		}
		if _, err := val2.Validate(file); err != nil {
			errg = err
		}
		if _, err := val3.Validate(file); err != nil {
			errg = err
		}
		if errg != nil {
			log.Printf("Validation failed")
			file.Close()
			os.Remove(path)
			w.Header().Set("Content-Type", " text/html")
			w.WriteHeader(200)
			fmt.Fprint(w, "Error. Your file must be a zip file")
			return
		}

		if buildproject {
			/*     -1 : Internal error while initialization template project
			1 : Cannot createTmpFile
			2 : Cannot copy template
			3 : Cannot copy unzip src to template copy
			4 : Cannot copy unzip resources to template copy
			5 : Cannot copy list all jar files
			6 : Cannot copy all jar files
			7 : Cannot generate maven pom.xml
			8 : Cannot execute maven
			9 : Cannot load surfire reports
			10 : Cannot load scalastype report
			11 : Cannot send an email
			*/
			resultNumber = 0

			tmpfolder, err := ioutil.TempDir("/tmp", binding.Username)
			//log.Print(tmpfolder)
			if err != nil {
				resultNumber = 1
				log.Print(err)

			}
			tmpfolder1, err := ioutil.TempDir("/tmp", binding.Username)
			//log.Print(tmpfolder1)
			if err != nil {
				resultNumber = 1
				log.Print(err)
			}

			//Step 1: Unzip upload file
			var tmpfolderpath, _ = filepath.Abs(tmpfolder)
			var tmpfolder1path, _ = filepath.Abs(tmpfolder1)
			//log.Printf(tmpfolderpath + "\n")
			//log.Printf(tmpfolder1path + "\n")

			Unzip(path, tmpfolderpath)

			log.Printf("Step 1 done\n")

			//Step 2: Copy template
			pathtmp, _ := filepath.Abs(templateProjectPath)
			err4 := copy.Copy(pathtmp, tmpfolder1path)
			if err4 != nil {
				resultNumber = 2
				log.Printf("Cannot copy %s\n", err4)
			}
			log.Printf("Step 2 done\n")

			//Step 3: identify folder
			files, _ := ioutil.ReadDir(tmpfolderpath)
			var f1 os.FileInfo
			for _, f := range files {
				if f.IsDir() {
					f1 = f
					break
				}
			}
			log.Printf("Step 3 done\n")

			//Step 4: Copy unzip src to template
			err4 = copy.Copy(tmpfolderpath+"/"+f1.Name()+"/src/", tmpfolder1path+"/src/main/scala/")
			if err4 != nil {
				resultNumber = 3
				log.Printf("Cannot copy %s\n", err4)
			}
			log.Printf("Step 4 done\n")

			//Step 6: Copy unzip resources to template
			if _, err2 := os.Stat(tmpfolderpath + "/" + f1.Name() + "/img"); err2 == nil {
				err4 = copy.Copy(tmpfolderpath+"/"+f1.Name()+"/img", tmpfolder1path+"/src/main/resources/img")
				if err4 != nil {
					resultNumber = 4
					log.Printf("Cannot copy %s\n", err4)
				}
				log.Printf("Step 6 done\n")
			}

			//Step 7: Copy jar to lib
			//history = child_process.execSync('cp -r '+ tmpfolder.name + '/'+dirProjet+'/*.jar '+tmpfolder1.name +'/lib/' , { encoding: 'utf8' });
			files2, err := filepath.Glob(tmpfolderpath + "/" + f1.Name() + "/*.jar")
			if err != nil {
				resultNumber = 5
				log.Print(err)
			}
			for _, jarfile := range files2 {
				if !copyFile(jarfile, tmpfolder1path+"/lib/"+filepath.Base(jarfile)) {
					resultNumber = 6
				}
			}
			log.Printf("Step 7 done\n")

			//Step 8 generate pom.xml
			//		var files = glob.sync(path.join(tmpfolder1.name  , '/lib/*.jar'));
			files1, err := filepath.Glob(tmpfolder1path + "/lib/*.jar")
			if err != nil {
				log.Fatal(err)
				resultNumber = 5

			}
			replacement := ""
			libn := 0
			for _, f := range files1 {
				replacement = replacement + "<dependency><artifactId>delfinelib" + fmt.Sprintf("%v", libn) + "</artifactId><groupId>delfinelib</groupId><version>1.0</version><scope>system</scope><systemPath>${project.basedir}/lib/" + filepath.Base(f) + "</systemPath></dependency>"
				libn = libn + 1
			}

			readfile, err3 := ioutil.ReadFile(tmpfolder1path + "/pom.xml")
			if err3 != nil {
				resultNumber = 6
				log.Print(err3)
			}

			newContents := strings.Replace(string(readfile), "<!--deps-->", replacement, -1)

			err3 = ioutil.WriteFile(tmpfolder1path+"/pom.xml", []byte(newContents), 0)
			if err3 != nil {
				resultNumber = 7
				log.Print(err3)
			}
			log.Printf("Step 8 done\n")

			//Step 9 Execute maven
			var stdout, stderr bytes.Buffer
			cmd := exec.Command(mavenhome+"/mvn", "-f", tmpfolder1path+"/pom.xml", "clean", "scalastyle:check", "test", "-Dmaven.test.failure.ignore=true")
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err1 := cmd.Run()
			if err1 != nil {

				resultNumber = 8

				log.Printf("cmd.Run() failed with %s\n", err1)
				fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
				fmt.Println(fmt.Sprint(err) + ": " + stdout.String())
			}
			_, errStr := string(stdout.Bytes()), string(stderr.Bytes())
			//_ = outStr

			//fmt.Printf("out:\n%s\nerr:\n%s\n", outStr, errStr)
			fmt.Printf("err:\n%s\n", errStr)
			log.Printf("Step 9 done\n")

			// Step 10 Check result and send them to IHM
			ntests := 0
			nerrors := 0
			nskips := 0
			nfailures := 0
			_ = ntests
			_ = nerrors
			_ = nskips
			_ = nfailures
			files2, err6 := filepath.Glob(tmpfolder1path + "/target/surefire-reports/*.xml")
			if err6 != nil {
				resultNumber = 9
				log.Print(err6)
			}
			for _, f3 := range files2 {
				f4, err := os.Open(f3)
				if err != nil {
					resultNumber = 9
				}
				doc, err := xmlquery.Parse(f4)
				if err != nil {
					resultNumber = 9
				}
				expr, err := xpath.Compile("count(//testsuite//testcase)")
				if err != nil {
					resultNumber = 9
				}
				ntests = ntests + int(expr.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64))
				expr1, err := xpath.Compile("count(//testsuite//testcase//failure)")
				if err != nil {
					resultNumber = 9
				}
				nfailures = nfailures + int(expr1.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64))
				expr2, err := xpath.Compile("count(//testsuite//testcase//error)")
				if err != nil {
					resultNumber = 9
				}
				nerrors = nerrors + int(expr2.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64))
				expr3, err := xpath.Compile("count(//testsuite//testcase//skipped)")
				if err != nil {
					resultNumber = 9
				}
				nskips = nskips + int(expr3.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64))

				fmt.Printf("nbtest: %d\n", ntests)
				_ = doc
			}

			f5, err := os.Open(tmpfolder1path + "/scalastyle-output.xml")
			if err != nil {
				resultNumber = 10
			}
			doc1, err := xmlquery.Parse(f5)
			if err != nil {
				resultNumber = 10
			}
			expr4, err := xpath.Compile("count(//checkstyle//file//error[@severity='warning'])")
			if err != nil {
				resultNumber = 10
			}
			warningstyle := int(expr4.Evaluate(xmlquery.CreateXPathNavigator(doc1)).(float64))
			expr5, err := xpath.Compile("count(//checkstyle//file//error[@severity='error'])")
			if err != nil {
				resultNumber = 10
			}
			errorstyle := int(expr5.Evaluate(xmlquery.CreateXPathNavigator(doc1)).(float64))

			if sendemail {
				mailaddr, err := getMail(binding.Username)

				if (err != nil) || (!sendEmail("Cher(e) "+binding.Username+"\n\nVous venez de charger un TP qui a été vérifié. L'archive est valide, le projet compile et vous avez "+fmt.Sprintf("%v", nerrors)+" test(s) en erreur, "+fmt.Sprintf("%v", nfailures)+" test(s) en échec, "+fmt.Sprintf("%v", nskips)+" non executé(s), sur un total de "+fmt.Sprintf("%v", ntests)+" tests.\nGardez trace de cet email en cas de litige pour l'upload. Le nom du fichier sur le serveur est le "+path+".\nBonne journée. \n\nSincèrement/ \nL'équipe pédagogique", "Rendu tp", mailaddr)) {
					resultNumber = 11
				}
			}
			//Step 12 remove tmpfolders
			defer os.RemoveAll(tmpfolderpath)
			defer os.RemoveAll(tmpfolder1path)
			/*
				1 : Cannot createTmpFile
				2 : Cannot copy template
				3 : Cannot copy unzip src to template copy
				4 : Cannot copy unzip resources to template copy
				5 : Cannot copy list all jar files
				6 : Cannot copy all jar files
				7 : Cannot generate maven pom.xml
				8 : Cannot execute maven
				9 : Cannot load surfire reports
				10 : Cannot load scalastype report
				11 : Cannot send an email
			*/
			switch resultNumber {
			case -1:
				fmt.Fprintf(w, "Internal error during the upload process")

			case 0:
				fmt.Fprintf(w, "nombre de tests executés : "+fmt.Sprintf("%v", ntests)+"<BR>"+"nombre de tests en erreur : "+fmt.Sprintf("%v", nerrors)+"<BR>"+"nombre de tests non exécutés : "+fmt.Sprintf("%v", nskips)+"<BR>"+"nombre de tests en échec : "+fmt.Sprintf("%v", nfailures)+"<BR>"+"nombre de style (scalastyle) en warning : "+fmt.Sprintf("%v", warningstyle)+"<BR>"+"nombre de style (scalastyle) en erreur : "+fmt.Sprintf("%v", errorstyle)+"<BR>")
			case 1:
				fmt.Fprintf(w, "Error during the build process <BR>(Cannot createTmpFile)<BR>")
			case 2:
				fmt.Fprintf(w, "Error during the build process <BR>(Cannot copy template)<BR>")
			case 3:
				fmt.Fprintf(w, "Error during the build process <BR>(Cannot copy unzip src to template copy)<BR>")
			case 4:
				fmt.Fprintf(w, "Error during the build process <BR>(Cannot copy unzip resources to template copy)<BR>")
			case 5:
				fmt.Fprintf(w, "Error during the build process <BR>(Cannot copy list all jar files)<BR>")
			case 6:
				fmt.Fprintf(w, "Error during the build process <BR>(Cannot copy all jar files)<BR>")
			case 7:
				fmt.Fprintf(w, "Error during the build process <BR>(Cannot generate maven pom.xml)<BR>")
			case 8:
				fmt.Fprintf(w, "Error during the build process <BR>(Cannot execute maven)<BR>")
			case 9:
				fmt.Fprintf(w, "Error during the build process <BR>(Cannot load or query surfire reports)<BR>")
			case 10:
				fmt.Fprintf(w, "Error during the build process <BR>(Cannot load or query scalastype report)<BR>")
			case 11:
				fmt.Fprintf(w, "Error. cannot send the email<BR>")
			default:
				fmt.Fprintf(w, "default result %v", resultNumber)
			}
		} else {
			if sendemail {
				mailaddr, err := getMail(binding.Username)

				if (err != nil) || (!sendEmail("Cher(e) "+binding.Username+"\n\nVous venez de charger un TP qui a été vérifié. L'archive est valide.\nGardez trace de cet email en cas de litige pour l'upload. Le nom du fichier sur le serveur est le "+path+".\nBonne journée. \n\nSincèrement/ \nL'équipe pédagogique", "Rendu tp", mailaddr)) {
					resultNumber = 11
				}
			}
			switch resultNumber {
			case -1:
				fmt.Fprintf(w, "Internal error during the upload process")
			case 0:
				fmt.Fprintf(w, "Upload réussi <BR>")
			case 11:
				fmt.Fprintf(w, "Error. Cannot send the email<BR>")
			}
		}

	}
}

func sendEmail(body string, subj string, tos string) bool {
	from := mail.Address{"Responsable L1 SI2", "resp-l1-si2@univ-rennes1.fr"}
	to := mail.Address{"", tos}
	log.Println(to)
	// Setup headers
	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = to.String()
	headers["Subject"] = subj

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	host, _, _ := net.SplitHostPort(smtpserver)

	auth := smtp.PlainAuth("", login, pass, host)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}

	// Here is the key, you need to call tls.Dial instead of smtp.Dial
	// for smtp servers running on 465 that require an ssl connection
	// from the very beginning (no starttls)
	c, err := smtp.Dial(smtpserver)
	if err != nil {
		log.Printf("could not connect to smtp server " + smtpserver)
		return false
	}

	c.StartTLS(tlsconfig)

	// Auth
	if err = c.Auth(auth); err != nil {
		log.Printf("Could not login to SMTP server")
		return false
	}

	// To && From
	if err = c.Mail(from.Address); err != nil {
		log.Printf("Bad from address")
		return false
	}

	if err = c.Rcpt(to.Address); err != nil {
		log.Printf("Bad to address")
		return false
	}

	// Data
	w, err := c.Data()
	if err != nil {
		//log.Panic(err)
		return false
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		//log.Panic(err)
		return false
	}

	err = w.Close()
	if err != nil {
		//log.Panic(err)
		return false
	}

	c.Quit()
	return true
}

func copyFile(src, dst string) bool {
	from, err := os.Open(src)
	if err != nil {
		log.Print(err)
		return false
	}
	defer from.Close()

	to, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Print(err)
		return false
		//		log.Fatal(err)
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		log.Print(err)
		return false
	}
	return true
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)
		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {

			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)

		} else {

			// Make File
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return filenames, err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return filenames, err
			}

			_, err = io.Copy(outFile, rc)

			// Close the file without defer to close before next iteration of loop
			outFile.Close()
			if err != nil {
				return filenames, err
			}

		}
	}
	return filenames, nil
}

func getMail(uid string) (string, error) {
	deft := "barais@irisa.fr"

	l, err := ldap.Dial("tcp", ldapserver) //fmt.Sprintf("%s:%d", "ldap.univ-rennes1.fr", 389))
	if err != nil {
		return deft, err
	}
	defer l.Close()
	// Reconnect with TLS
	err = l.StartTLS(&tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return deft, err
	}

	//15010426
	searchRequest := ldap.NewSearchRequest(
		"ou=People,dc=univ-rennes1,dc=fr", // The base dn to search
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(&(uid="+uid+"))", // The filter to apply
		[]string{"mail"},   // A list attributes to retrieve
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return deft, err
	}

	for _, entry := range sr.Entries {
		return entry.GetAttributeValue("mail"), nil
		//fmt.Printf("%s: %v\n", entry.DN, entry.GetAttributeValue("mail"))
	}
	//	fmt.Println("ok")
	return deft, errors.New("getMail : couldn't find student email address")
}
