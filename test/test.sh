
#!/bin/bash

# Usage: cas-get.sh {url} {username} {password} # If you have any errors try removing the redirects to get more information
# The service to be called, and a url-encoded version (the url encoding isn't perfect, if you're encoding complex stuff you may wish to replace with a different method)
DEST=$3
ENCODED_DEST=`echo "$DEST" | perl -p -e 's/([^A-Za-z0-9])/sprintf("%%%02X", ord($1))/seg' | sed 's/%2E/./g' | sed 's/%0A//g'`

#IP Addresses or hostnames are fine here
CAS_HOSTNAME="sso-cas.univ-rennes1.fr"

#Authentication details. This script only supports username/password login, but curl can handle certificate login if required
USERNAME=$1
PASSWORD=$2

#Temporary files used by curl to store cookies and http headers
COOKIE_JAR=.cookieJar
HEADER_DUMP_DEST=.headers
rm $COOKIE_JAR
rm $HEADER_DUMP_DEST

#The script itself is below

#Visit CAS and get a login form. This includes a unique ID for the form, which we will store in CAS_ID and attach to our form submission. jsessionid cookie will be set here
#CAS_ID=`curl -s -k -c ${COOKIE_JAR} https://${CAS_HOSTNAME}/cas/login?service=${ENCODED_DEST} | tee response1.txt | grep 'name=.execution\|name=.lt' | sed 's/.*value..//' | sed 's/\".*//'`
CAS_ID=`curl -s -k -c $COOKIE_JAR https://$CAS_HOSTNAME/login?service=$ENCODED_DEST | grep name=.lt | sed 's/.*value..//' | sed 's/\".*//'`

if [[ "$CAS_ID" = "" ]]; then
   echo "Login ticket is empty."
   exit 1
fi

#Submit the login form, using the cookies saved in the cookie jar and the form submission ID just extracted. We keep the headers from this request as the return value should be a 302 including a "ticket" param which we'll need in the next request
curl -s -k --data "username=$USERNAME&password=$PASSWORD&lt=$CAS_ID&execution=e1s1&_eventId=submit" -i -b $COOKIE_JAR -c $COOKIE_JAR https://$CAS_HOSTNAME/login?service=$ENCODED_DEST -D $HEADER_DUMP_DEST -o /dev/null

#Linux may not need this line but my response from the previous call has retrieving windows-style linebreaks in OSX
#dos2unix $HEADER_DUMP_DEST > /dev/null

#Visit the URL with the ticket param to finally set the casprivacy and, more importantly, MOD_AUTH_CAS cookie. Now we've got a MOD_AUTH_CAS cookie, anything we do in this session will pass straight through CAS
CURL_DEST=`grep Location $HEADER_DUMP_DEST | sed 's/Location: //'`

if [[ "$CURL_DEST" = "" ]]; then
    echo "Cannot login. Check if you can login in a browser using user/pass = $USERNAME/$PASSWORD and the following url: https://$CAS_HOSTNAME/cas/login?service=$ENCODED_DEST"
    exit 1
fi

#echo $COOKIE_JAR 

echo $CURL_DEST
CURL_DEST=${CURL_DEST%$'\r'}


#curl -k -b $COOKIE_JAR -c $COOKIE_JAR $CURL_DEST
#curl -b $COOKIE_JAR  $CURL_DEST

curl \
  -i -X POST -H "Content-Type: multipart/form-data"  \
  -F "uploads=@$4" \
  $CURL_DEST

#If our destination is not a GET we'll need to do a GET to, say, the user dashboard here

#Visit the place we actually wanted to go to
#curl -s -k -b $COOKIE_JAR "$DEST"
