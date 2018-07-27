package main

import (
	"os/exec"
    "archive/zip"
	"strings"
	"io"
	"bytes"
	"io/ioutil"
	"net/http"
	"net"
	"flag"
	"log"
	"os"
	"fmt"
	"net/mail"
	"net/smtp"
	"path/filepath"
	"crypto/tls"
	"github.com/thanhpk/randstr"
    //"html/template"
    "net/url"
    "gopkg.in/cas.v2"
	"github.com/adelowo/filer"
	"github.com/adelowo/filer/validator"
	"github.com/otiai10/copy"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/barais/ipfilter"
    // TODO extend with time
	//	"github.com/jpillora/ipfilter"
)

//TODO 
// Filter by IP and time





var fileserver http.Handler
var login string
var pass string
var uploadfolder string
var mavenhome string
var templateProjectPath string


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
	flag.Parse()
	fileserver	= http.FileServer(http.Dir(*directory))
	login = *paramlogin
	pass = *parampass
	uploadfolder = *uploadfolderparam
	mavenhome = *mavenhomeparam
	templateProjectPath = *templateProjectPathparam

	mux := http.NewServeMux();

	mux.HandleFunc("/", testCas)
	url, _ := url.Parse(*casURL)
    client := cas.NewClient(&cas.Options{
        URL: url,
	})
	myProtectedHandler := ipfilter.Wrap(client.Handle(mux), ipfilter.Options{
		//block requests from China and Russia by IP
		BlockedCountries: []string{"CN", "RU"},
	})
	
	log.Printf("Serving %s on HTTP port: %s\n", *directory, *port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+*port,myProtectedHandler ))

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
    binding := &templateBinding{
        Username:   cas.Username(r),
        Attributes: cas.Attributes(r),
	}
	//log.Printf("username: %s\n", binding.Username)
	//log.Printf("ip: %s\n", 	getRealAddr(r))
	if r.URL.Path == "/upload" {
        uploadProgress(w, r,binding)
        return
	}

	fileserver.ServeHTTP(w,r)
}


func uploadProgress(w http.ResponseWriter, r *http.Request, binding *templateBinding) {
	mr, err := r.MultipartReader()
    if err != nil {
        return
    }
//    length := r.ContentLength
    for {
        part, err := mr.NextPart()
        if err == io.EOF {
            break
		}
		token := randstr.String(24) // generate a random 16 character length string
        var read int64
//		var p float32
		path := uploadfolder+ "/"+ binding.Username+"_"+token +".zip"
        dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
        if err != nil {
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
			fmt.Fprint(w, "Your file must be a zip file")
			return
		}

		tmpfolder, err := ioutil.TempDir("/tmp", binding.Username)
		if err != nil {
			log.Fatal(err)
		}
		tmpfolder1, err := ioutil.TempDir("/tmp", binding.Username)
		if err != nil {
			log.Fatal(err)
		}

		//Step 1: Unzip upload file
		var tmpfolderpath,_ =filepath.Abs(tmpfolder) 
		var tmpfolder1path,_ =filepath.Abs(tmpfolder1) 
		log.Printf(tmpfolderpath+"\n")
		log.Printf(tmpfolder1path+"\n")

		var stdout, stderr bytes.Buffer
		/*cmd := exec.Command("unzip", path,"-d",tmpfolderpath)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err1 := cmd.Run()
		if err1 != nil {
			log.Fatalf("cmd.Run() failed with %s\n", err1)
		}
		outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
		_ = outStr
		//fmt.Printf("out:\n%s\nerr:\n%s\n", outStr, errStr)
		fmt.Printf("err:\n%s\n", errStr)*/

		Unzip(path,tmpfolderpath)
		
		log.Printf("Step 1 done\n")

		pathtmp,_ :=filepath.Abs(templateProjectPath)
		//Step 2: Copy template		
		err4 := copy.Copy(pathtmp, tmpfolder1path)
		if err4 != nil {
			log.Fatalf("Cannot copy %s\n", err4)
		}
	//	log.Printf("Step 2 done\n")


		//Step 3: identify folder
		files, _ := ioutil.ReadDir(tmpfolderpath)
		var f1 os.FileInfo
		for _, f := range files {
			if f.IsDir(){
				f1 = f
				break
			}
		}
	//	log.Printf("Step 3 done\n")

		//Step 4: Copy unzip src to template
		err4 = copy.Copy(tmpfolderpath+"/" + f1.Name()+"/src/", tmpfolder1path+"/src/main/scala/")
		if err4 != nil {
			log.Fatalf("Cannot copy %s\n", err4)
		}
	//	log.Printf("Step 4 done\n")


		//Step 6: Copy unzip resources to template
		if _, err2 := os.Stat(tmpfolderpath + "/"+f1.Name()+"/img"); err2 == nil {
			err4 = copy.Copy(tmpfolderpath+"/" + f1.Name()+"/img", tmpfolder1path+"/src/main/resources/img")
			if err4 != nil {
				log.Fatalf("Cannot copy %s\n", err4)
			}
				log.Printf("Step 6 done\n")
		}


		//Step 7: Copy jar to lib
		//history = child_process.execSync('cp -r '+ tmpfolder.name + '/'+dirProjet+'/*.jar '+tmpfolder1.name +'/lib/' , { encoding: 'utf8' });
		files2, err := filepath.Glob(tmpfolderpath+"/" + f1.Name()+"/*.jar")
		if err != nil {
    	    log.Fatal(err)
		}
        for _, jarfile := range files2 {
				CopyFile(jarfile,tmpfolder1path +"/lib/"+filepath.Base(jarfile))		
		}
//		log.Printf("Step 7 done\n")

		//Step 8 generate pom.xml
//		var files = glob.sync(path.join(tmpfolder1.name  , '/lib/*.jar'));
		files1, err := filepath.Glob(tmpfolder1path+"/lib/*.jar")
	    if err != nil {
    	    log.Fatal(err)
		}
        replacement := ""
        libn := 0
		for _, f := range files1 {
			replacement = replacement+ "<dependency><artifactId>delfinelib"+fmt.Sprintf("%v",libn)+"</artifactId><groupId>delfinelib</groupId><version>1.0</version><scope>system</scope><systemPath>${project.basedir}/lib/"+filepath.Base(f)+"</systemPath></dependency>"
			libn = libn+1;
		  }
		  

		  readfile, err3 := ioutil.ReadFile(tmpfolder1path + "/pom.xml")
		  if err3 != nil {
			  panic(err3)
		  }
  
		  newContents := strings.Replace(string(readfile), "<!--deps-->", replacement, -1)

		  err3 = ioutil.WriteFile(tmpfolder1path + "/pom.xml", []byte(newContents), 0)
		  if err3 != nil {
			  panic(err3)
		  }  
//		  log.Printf("Step 8 done\n")

		//Step 9 Execute maven
		//          var history = child_process.execSync(mavenhome + '/bin/mvn -f'+ tmpfolder1.name + '/pom.xml clean scalastyle:check test -Dmaven.test.failure.ignore=true' , { encoding: 'utf8' });
		cmd := exec.Command(mavenhome + "/mvn", "-f",tmpfolder1path+ "/pom.xml", "clean", "scalastyle:check", "test", "-Dmaven.test.failure.ignore=true" )
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err1 := cmd.Run()
		if err1 != nil {
			log.Fatalf("cmd.Run() failed with %s\n", err1)
		}
		outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
		_ = outStr

		//fmt.Printf("out:\n%s\nerr:\n%s\n", outStr, errStr)
		fmt.Printf("err:\n%s\n", errStr)
//		log.Printf("Step 9 done\n")


		// Step 10 Check result and send them to IHM
		ntests := 0
        nerrors := 0
        nskips :=0
		nfailures := 0
		_= ntests
		_= nerrors
		_= nskips
		_= nfailures
		files2, err6 := filepath.Glob(tmpfolder1path+"/target/surefire-reports/*.xml")
	    if err6 != nil {
    	    log.Fatal(err6)
		}
		for _, f3 := range files2 {
			f4, _ := os.Open(f3)
			doc, _ := xmlquery.Parse(f4)
			expr, _ := xpath.Compile("count(//testsuite//testcase)")
			ntests = ntests +int(expr.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64))
			expr1, _ := xpath.Compile("count(//testsuite//testcase//failure)")
			nfailures = nfailures +int(expr1.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64))
			expr2, _ := xpath.Compile("count(//testsuite//testcase//error)")
			nerrors = nerrors +int(expr2.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64))
			expr3, _ := xpath.Compile("count(//testsuite//testcase//skipped)")
			nskips = nskips +int(expr3.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64))

			fmt.Printf("nbtest: %d\n", ntests)
			_ = doc
		}
 
		f5, _ := os.Open(tmpfolder1path  + "/scalastyle-output.xml") ;
		doc1, _ := xmlquery.Parse(f5)
		expr4, _ := xpath.Compile("count(//checkstyle//file//error[@severity='warning'])")
		warningstyle := int(expr4.Evaluate(xmlquery.CreateXPathNavigator(doc1)).(float64))
		expr5, _ := xpath.Compile("count(//checkstyle//file//error[@severity='error'])")
		errorstyle := int(expr5.Evaluate(xmlquery.CreateXPathNavigator(doc1)).(float64))
		


		 sendEmail("Cher(e) " + binding.Username + "\n\nVous venez de charger un TP qui a été vérifié. L'archive est valide, le projet compile et vous avez " + fmt.Sprintf("%v",nerrors) + " test(s) en erreur, "+ fmt.Sprintf("%v",nfailures)  + " test(s) en échec, " + fmt.Sprintf("%v",nskips) + " non executé(s), sur un total de "+ fmt.Sprintf("%v",ntests) +" tests.\nGardez trace de cet email en cas de litige pour l'upload. Le nom du fichier sur le serveur est le " + path+".\nBonne journée. \n\nSincèrement/ \nL'équipe pédagogique","Rendu tp",binding.Username+"@univ-rennes1.fr")

		//Step 12 remove tmpfolders
		defer os.RemoveAll(tmpfolderpath)
		defer os.RemoveAll(tmpfolder1path)

		fmt.Fprintf(w, "nombre de tests executés : " +  fmt.Sprintf("%v",ntests) +"<BR>"+ "nombre de tests en erreur : " + fmt.Sprintf("%v",nerrors) +"<BR>"+"nombre de tests non exécutés : " +  fmt.Sprintf("%v",nskips) +"<BR>"+ "nombre de tests en échec : " +  fmt.Sprintf("%v",nfailures) +"<BR>"+ "nombre de style (scalastyle) en warning : " + fmt.Sprintf("%v",warningstyle) +"<BR>"+"nombre de style (scalastyle) en erreur : " +  fmt.Sprintf("%v",errorstyle) +"<BR>");
    }	
}





func sendEmail(body string,subj string,tos string) bool{
	from := mail.Address{"Olivier Barais", "obarais@univ-rennes1.fr"}
    to   := mail.Address{"", tos}

    // Setup headers
    headers := make(map[string]string)
    headers["From"] = from.String()
    headers["To"] = to.String()
    headers["Subject"] = subj

    // Setup message
    message := ""
    for k,v := range headers {
        message += fmt.Sprintf("%s: %s\r\n", k, v)
    }
    message += "\r\n" + body

    // Connect to the SMTP Server
    servername := "smtps.univ-rennes1.fr:587"

    host, _, _ := net.SplitHostPort(servername)

    auth := smtp.PlainAuth("",login, pass, host)

    // TLS config
    tlsconfig := &tls.Config {
        InsecureSkipVerify: true,
        ServerName: host,
    }

    // Here is the key, you need to call tls.Dial instead of smtp.Dial
    // for smtp servers running on 465 that require an ssl connection
    // from the very beginning (no starttls)
	c, err := smtp.Dial(servername)
    if err != nil {
        log.Panic(err)
    }

    c.StartTLS(tlsconfig)

    // Auth
    if err = c.Auth(auth); err != nil {
        log.Panic(err)
		return false
    }

    // To && From
    if err = c.Mail(from.Address); err != nil {
        log.Panic(err)
		return false
    }

    if err = c.Rcpt(to.Address); err != nil {
        log.Panic(err)
		return false
    }

    // Data
    w, err := c.Data()
    if err != nil {
        log.Panic(err)
		return false
    }

    _, err = w.Write([]byte(message))
    if err != nil {
        log.Panic(err)
		return false
    }

    err = w.Close()
    if err != nil {
        log.Panic(err)
		return false
    }

	c.Quit()
	return true
}

func CopyFile(src, dst string)  {
	from, err := os.Open(src)
	if err != nil {
	  log.Fatal(err)
	}
	defer from.Close()
  
	to, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
	  log.Fatal(err)
	}
	defer to.Close()
  
	_, err = io.Copy(to, from)
	if err != nil {
	  log.Fatal(err)
	}
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