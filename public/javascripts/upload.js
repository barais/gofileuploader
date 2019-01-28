

$('.upload-btn').on('click', function (){
    $('#upload-input').click();
    $('.progress-bar').text('0%');
    $('.progress-bar').width('0%');
});

$('#upload-input').on('change', function(){

  var files = $(this).get(0).files;

  if (files.length > 0){
    $("#myModal1").modal({backdrop: 'static', keyboard: false});

    // create a FormData object which will be sent as the data payload in the
    // AJAX request
    var formData = new FormData();

    // loop through all the selected files and add them to the formData object
    for (var i = 0; i < files.length; i++) {
      var file = files[i];

      // add the files to formData object for the data payload
      formData.append('uploads[]', file, file.name);
    }

    $.ajax({
      url: '/upload',
      type: 'POST',
      data: formData,
      processData: false,
      contentType: false,
      success: function(data){
          console.log('upload successful!\n' + data);
          if (data == 'Your file must be a zip file'){
              $('#progress-bar-upload1').text('0' + '%');
              $('#progress-bar-upload1').width('0' + '%');
              $('#progress-bar-upload').html('Dépôt échoué');
              $('#progress-bar-test').html('');

              $('#mymodalbody').html(
                  "<div class=\"alert alert-danger\" role=\"alert\">"
                      + "<div class=\"row\">"
                      + "   <div class=\"col-md-2\">"
                      +       "<span class=\"glyphicon glyphicon-exclamation-sign\" aria-hidden=\"true\" style=\"color:red\"></span>"
                      + "   </div>"
                      + "   <div class=\"col-md-10\">"
                      +       '<p>'+ "Le TP déposé doit être une archive ZIP. <BR> <BR>"
                      +       "Exportez votre projet Eclipse au format ZIP <BR>"
                      +       "et recommencez le dépôt." +'</p>'
                      + "   </div>"
                      + "</div>"
                 + "</div>"
              );
              $("#myModal1").modal('hide');
              $("#myModal").modal();
          }
          else if (data.toLowerCase().indexOf("error") !== -1) {
              $('#progress-bar-upload1').text('0' + '%');
              $('#progress-bar-upload1').width('0' + '%');
              $('#progress-bar-upload').html('Dépôt effectué');
              $('#progress-bar-test').html('Validation échouée');

              $('#mymodalbody').html(
                  "<div class=\"alert alert-danger\" role=\"alert\">"
                      + "<div class=\"row\">"
                      + "   <div class=\"col-md-2\">"
                      +       "<span class=\"glyphicon glyphicon-exclamation-sign\" aria-hidden=\"true\" style=\"color:red\"></span>"
                      + "   </div>"
                      + "   <div class=\"col-md-10\">"
                      +       '<p>' + "Le dépôt est réalisé, mais la validation a échoué" + '</p>'
                      +       '<p>' + data.replace(/\n/g, "<BR>") + '</p>'
                      +       '<p>' + "Corrigez ces problèmes, <BR> puis ré-exportez votre projet au format ZIP <BR> "
                      +               "puis recommencez le dépôt." + '</p>'
                      + "   </div>"
                      + "</div>"
                      + "</div>"
              );
              $("#myModal1").modal('hide');
              $("#myModal").modal();
          }
          else{
              $('#progress-bar-test1').text('0' + '%');
              $('#progress-bar-test1').width('0' + '%');
              $('#progress-bar-upload').html('Dépôt effectué');
              $('#progress-bar-test').html('Validation zip réalisée.');
              $('#mymodalbody').html(
                  "<div class=\"alert alert-success\" role=\"alert\">"
                      + "<div class=\"row\">"
                      + "   <div class=\"col-md-2\">"
                      +       "<span class=\"glyphicon glyphicon-ok\" aria-hidden=\"true\" style=\"color:green\"></span>"
                      + "   </div>"
                      + "   <div class=\"col-md-10\">"
                      +       '<p>' + data.replace(/\n/g, "<BR>") + '</p>'
                      + "   </div>"
                      + "</div>"
                      + "</div>");

              $("#myModal1").modal('hide');
              $("#myModal").modal();
          }
      },
      error : function(resultat, statut, erreur){
        $("#myModal1").modal('hide');
          window.alert("Erreur interne. Veuillez recommencer.\n"
                       + "(Log: " + resultat + "\n" + statut + "\n" + erreur + ")");
          $('#progress-bar-upload').html('');
          $('#progress-bar-test').html('');

      },
      xhr: function() {
        // create an XMLHttpRequest
        var xhr = new XMLHttpRequest();

        // listen to the 'progress' event
        xhr.upload.addEventListener('progress', function(evt) {

          if (evt.lengthComputable) {
            // calculate the percentage of upload completed
            var percentComplete = evt.loaded / evt.total;
            percentComplete = parseInt(percentComplete * 100);
            // update the Bootstrap progress bar with the new percentage
            $('#progress-bar-upload1').text(percentComplete + '%');
            $('#progress-bar-upload1').width(percentComplete + '%');

            // once the upload reaches 100%, set the progress bar text to done
            if (percentComplete === 100) {
              $('#progress-bar-upload').html('Dépôt effectué');
              $('#progress-bar-test').html('');
            }

          }

        }, false);

        return xhr;
      }
    });

  }
});
