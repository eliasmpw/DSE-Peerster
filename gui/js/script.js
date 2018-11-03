$(document).ready(function () {
    getId();
    getMessages();
    getPeerNodes();
    getAllNodes();
    // Update data with 1 or 2 seconds ticks
    setInterval(function () {
        getMessages();
    }, 1000);
    setInterval(function () {
        getPeerNodes();
    }, 2000);
    setInterval(function () {
        getAllNodes();
    }, 2000);

    $('#sendMessage').click(postMessage);
    $('#addPeerNode').click(postPeerNode);
    $('#newMessage').keyup(enableSendMessageBtn);
    $('#newPeerNode').keyup(enableNewPeerNodeBtn);
    $('#privateMessageModal').on('show.bs.modal', onModalOpened);

    let selectedPrivateName;
    let selectedPrivateAddress;
    let privateMessageInterval;

    function getMessages() {
        let messagesElement = $('#messages');
        $.ajax({
            type: 'GET',
            url: '/message',
            data: '',
            success: function (response) {
                if (response) {
                    const oldValue = messagesElement.text();
                    messagesElement.text('');
                    $.each(response, function (key, value) {
                        messagesElement.append(sanitizeString(value.Origin) + ': ' +
                            sanitizeString(value.Text) + '\n');
                    });
                    if (oldValue !== messagesElement.text()) {
                        messagesElement.scrollTop(messagesElement.prop('scrollHeight'));
                    }
                }
            }
        });
    }

    function postMessage() {
        const newMessage = $('#newMessage').val();
        if (newMessage != '') {
            const GossipPacket = {
                Simple: {
                    OriginalName: '',
                    RelayPeerAddr: '',
                    Contents: newMessage,
                }
            };
            const packet = JSON.stringify(GossipPacket);
            console.log(packet);
            $.ajax({
                type: 'POST',
                url: '/message',
                data: packet,
            });
        }
        $('#newMessage').val('');
        enableSendMessageBtn();
    }

    function getPeerNodes() {
        let nodesElement = $('#peerNodes');
        $.ajax({
            type: 'GET',
            url: '/node',
            data: '',
            success: function (response) {
                if (response) {
                    const oldValue = nodesElement.text();
                    nodesElement.text('');
                    $.each(response, function (index, value) {
                        nodesElement.append(sanitizeString(value) + '\n');
                    });
                    if (oldValue !== nodesElement.text()) {
                        nodesElement.scrollTop(nodesElement.prop('scrollHeight'));
                    }
                }
            }
        });
    }

    function postPeerNode() {
        const newPeerNode = $('#newPeerNode').val();
        if (isValidIPWithPort(newPeerNode)) {
            const GossipPacket = {
                Node: newPeerNode
            };
            const packet = JSON.stringify(GossipPacket);
            $.ajax({
                type: 'POST',
                url: '/node',
                data: newPeerNode,
            });
        }
        $('#newPeerNode').val('');
        enableNewPeerNodeBtn();
    }

    function getId() {
        let idElement = $('#peerId');
        $.ajax({
            type: 'GET',
            url: '/id',
            data: '',
            success: function (response) {
                if (response) {
                    idElement.val(response);
                }
            }
        });
    }

    function isValidIPWithPort(value) {
        let parts = value.split(":");
        let ipAddress = parts[0].split(".");
        let port = parts[1];
        return isBetween(port, 1, 65535) && ipAddress.length === 4 && ipAddress.every(function (segment) {
            return isBetween(segment, 0, 255);
        });
    }

    function isBetween(value, min, max) {
        const num = +value;
        return num >= min && num <= max;
    }

    function sanitizeString(myString) {
        return myString
            .replace(/&/g, '&amp;')
            .replace(/>/g, '&gt;')
            .replace(/</g, '&lt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }

    function enableSendMessageBtn() {
        if ($('#newMessage').val() == '') {
            $('#sendMessage').prop('disabled', true);
        } else {
            $('#sendMessage').prop('disabled', false);
        }
    }

    function enableNewPeerNodeBtn() {
        if (isValidIPWithPort($('#newPeerNode').val())) {
            $('#addPeerNode').prop('disabled', false);
        } else {
            $('#addPeerNode').prop('disabled', true);
        }
    }

    function getAllNodes() {
        let nodesElement = $('#allNodes');
        $.ajax({
            type: 'GET',
            url: '/allNodes',
            data: '',
            success: function (response) {
                if (response) {
                    for (let knownNode in response) {
                        nodesElement.html('<div class="knownNode" ' +
                            'data-toggle="modal" data-target="#privateMessageModal" ' +
                            'data-name="' + knownNode + '" ' +
                            'data-address="' + response[knownNode] + '">' + knownNode + '</div>');
                    }
                }
            }
        });
    }

    function getPrivateMessages() {
        let messagesElement = $('#privateMessages');
        $.ajax({
            type: 'GET',
            url: '/privateMessages',
            data: '',
            success: function (response) {
                if (response) {
                    console.log(response);
                    console.log(selectedPrivateAddress);
                    console.log(selectedPrivateName);
                    // const oldValue = messagesElement.text();
                    // messagesElement.text('');
                    // $.each(response, function (key, value) {
                    //     messagesElement.append(sanitizeString(value.Origin) + ': ' +
                    //         sanitizeString(value.Text) + '\n');
                    // });
                    // if (oldValue !== messagesElement.text()) {
                    //     messagesElement.scrollTop(messagesElement.prop('scrollHeight'));
                    // }
                }
            }
        });
    }

    function onModalOpened(event) {
        selectedPrivateName = event.relatedTarget.dataset.name;
        selectedPrivateAddress = event.relatedTarget.dataset.address;
        getPrivateMessages();
        privateMessageInterval = setInterval(function () {
            getPrivateMessages();
        }, 1000);
    }
});