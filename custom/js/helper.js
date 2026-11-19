// escapeHtml makes a value safe to insert into HTML text and quoted attributes.
function escapeHtml(value) {
    if (value === undefined || value === null) {
        return "";
    }
    return String(value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

function renderClientList(data) {
    const e = escapeHtml;
    $.each(data, function(index, obj) {
        // render client status css tag style
        let clientStatusHtml = '>'
        if (obj.Client.enabled) {
            clientStatusHtml = `style="visibility: hidden;">`
        }

        // render client allocated ip addresses
        let allocatedIpsHtml = "";
        $.each(obj.Client.allocated_ips, function(index, obj) {
            allocatedIpsHtml += `<small class="badge badge-secondary">${e(obj)}</small>&nbsp;`;
        })

        // render client allowed ip addresses
        let allowedIpsHtml = "";
        $.each(obj.Client.allowed_ips, function(index, obj) {
            allowedIpsHtml += `<small class="badge badge-secondary">${e(obj)}</small>&nbsp;`;
        })

        let subnetRangesString = "";
        if (obj.Client.subnet_ranges && obj.Client.subnet_ranges.length > 0) {
            subnetRangesString = obj.Client.subnet_ranges.join(',')
        }

        let additionalNotesHtml = "";
        if (obj.Client.additional_notes && obj.Client.additional_notes.length > 0) {
            additionalNotesHtml = `<span class="info-box-text" style="display: none"><i class="fas fa-additional_notes"></i>${e(obj.Client.additional_notes.toUpperCase())}</span>`
        }

        const id = e(obj.Client.id);
        const name = e(obj.Client.name);

        // render client html content
        let html = `<div class="col-sm-6 col-md-6 col-lg-4" id="client_${id}">
                        <div class="info-box">
                            <div class="overlay" id="paused_${id}"` + clientStatusHtml
                                + `<i class="paused-client fas fa-3x fa-play" onclick="resumeClient('${id}')"></i>
                            </div>
                            <div class="info-box-content" style="overflow: hidden">
                                <div class="btn-group">
                                    <a href="download?clientid=${id}" class="btn btn-outline-primary btn-sm">Download</a>
                                </div>
                                <div class="btn-group">
                                    <button type="button" class="btn btn-outline-primary btn-sm" data-toggle="modal"
                                        data-target="#modal_qr_client" data-clientid="${id}"
                                        data-clientname="${name}" ${obj.QRCode != "" ? '' : ' disabled'}>QR code</button>
                                </div>
                                <div class="btn-group">
                                    <button type="button" class="btn btn-outline-danger btn-sm">More</button>
                                    <button type="button" class="btn btn-outline-danger btn-sm dropdown-toggle dropdown-icon"
                                        data-toggle="dropdown">
                                    </button>
                                    <div class="dropdown-menu" role="menu">
                                        <a class="dropdown-item" href="#" data-toggle="modal"
                                        data-target="#modal_edit_client" data-clientid="${id}"
                                        data-clientname="${name}">Edit</a>
                                        <a class="dropdown-item" href="#" data-toggle="modal"
                                        data-target="#modal_pause_client" data-clientid="${id}"
                                        data-clientname="${name}">Disable</a>
                                        <a class="dropdown-item" href="#" data-toggle="modal"
                                        data-target="#modal_remove_client" data-clientid="${id}"
                                        data-clientname="${name}">Delete</a>
                                    </div>
                                </div>
                                <hr>
                                <span class="info-box-text"><i class="fas fa-user"></i> ${name}</span>
                                <span class="info-box-text" style="display: none"><i class="fas fa-key"></i> ${e(obj.Client.public_key)}</span>
                                <span class="info-box-text" style="display: none"><i class="fas fa-subnetrange"></i>${e(subnetRangesString)}</span>
                                ${additionalNotesHtml}
                                <span class="info-box-text"><i class="fas fa-envelope"></i> ${e(obj.Client.email)}</span>
                                <span class="info-box-text"><i class="fas fa-clock"></i>
                                    ${prettyDateTime(obj.Client.created_at)}</span>
                                <span class="info-box-text"><i class="fas fa-history"></i>
                                    ${prettyDateTime(obj.Client.updated_at)}</span>
                                <span class="info-box-text"><i class="fas fa-server" style="${obj.Client.use_server_dns ? "opacity: 1.0" : "opacity: 0.5"}"></i>
                                    ${obj.Client.use_server_dns ? 'DNS enabled' : 'DNS disabled'}</span>
                                <span class="info-box-text"><i class="fas fa-file"></i>
                                    ${e(obj.Client.additional_notes)}</span>
                                <span class="info-box-text"><strong>IP Allocation</strong></span>`
                                + allocatedIpsHtml
                                + `<span class="info-box-text"><strong>Allowed IPs</strong></span>`
                                + allowedIpsHtml
                            +`</div>
                        </div>
                    </div>`

        // add the client html elements to the list
        $('#client-list').append(html);
    });
}

function renderUserList(data) {
    const e = escapeHtml;
    $.each(data, function(index, obj) {
        const username = e(obj.username);

        // render user html content
        let html = `<div class="col-sm-6 col-md-6 col-lg-4" id="user_${username}">
                        <div class="info-box">
                            <div class="info-box-content">
                                <div class="btn-group">
                                     <button type="button" class="btn btn-outline-primary btn-sm" data-toggle="modal" data-target="#modal_edit_user" data-username="${username}">Edit</button>
                                </div>
                                <div class="btn-group">
                                    <button type="button" class="btn btn-outline-danger btn-sm" data-toggle="modal"
                                        data-target="#modal_remove_user" data-username="${username}">Delete</button>
                                </div>
                                <hr>
                                <span class="info-box-text"><i class="fas fa-user"></i> ${username}</span>
                                <span class="info-box-text"><i class="fas fa-terminal"></i> ${obj.admin? 'Administrator':'Manager'}</span>
                                </div>
                        </div>
                    </div>`

        // add the user html elements to the list
        $('#users-list').append(html);
    });
}


function prettyDateTime(timeStr) {
    const dt = new Date(timeStr);
    const offsetMs = dt.getTimezoneOffset() * 60 * 1000;
    const dateLocal = new Date(dt.getTime() - offsetMs);
    return dateLocal.toISOString().slice(0, 19).replace(/-/g, "/").replace("T", " ");
}
