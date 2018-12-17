

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
            $("#myModal1").modal('hide');
            window.alert("Your file must be a zip file");
          }else{
            $('#progress-bar-test1').text('0' + '%');
            $('#progress-bar-test1').width('0' + '%');
            $('#progress-bar-upload').html('done');
            $('#progress-bar-test').html('done');
            console.log($('#myModal'));
            $('#mymodalbody').html('<p>'+data.replace(/\n/g, "<br />")+'</p>');
            $("#myModal1").modal('hide');

            $("#myModal").modal();

 //            window.alert(data);
          }
      },
      error : function(resultat, statut, erreur){
        $("#myModal1").modal('hide');
        window.alert("Upload error");
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
              $('#progress-bar-upload').html('Done');
              $('#progress-bar-test').html('Executing test');
            }

          }

        }, false);

        return xhr;
      }
    });

  }
});
