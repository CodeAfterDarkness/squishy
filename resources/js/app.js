console.log("app loaded");

function loadWO(id){
	$.ajax({
		url: "../wo/" + id,
		success: function(json){
			//console.log(json);

			$('#woID').val(json.ID);
			$('#woName').val(json.OrderName);
			$('#woSummary').val(json.Summary);
			$('#woDetails').val(json.Details);
			$('#woCustomerName').val(json.CustomerName);

		}
	})
}
