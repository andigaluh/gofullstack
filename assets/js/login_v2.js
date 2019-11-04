/* Webarch Admin Dashboard 
-----------------------------------------------------------------*/ 
$(document).ready(function() {		
	$('#login_toggle').click(function(){
		//$('#frm_login').show();
		//$('#frm_register').hide();
	})
	$('#register_toggle').click(function(){
		//$('#frm_login').hide();
		//$('#frm_register').show();
	})
	
	$(".lazy").lazyload({
      effect : "fadeIn"
   });
	
});