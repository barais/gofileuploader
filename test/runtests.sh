liste_fichiers=`ls $1`
folder=`pwd`
for fichier in $liste_fichiers
do
#	echo $fichier
	CURL_DEST=${fichier%$'\r'}
	echo "$folder/runworker.sh $2 $3 $4 $1/$CURL_DEST"
	sem -j 4 $folder/runworker.sh \"$2\" \"$3\" $4 $1/$CURL_DEST 
done
sem --wait 
rm -rf /tmp/si2*
