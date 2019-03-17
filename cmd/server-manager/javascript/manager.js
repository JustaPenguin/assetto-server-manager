"use strict";

let $document;

// entry-point
$(document).ready(function () {
    console.log("initialising server manager javascript");

    $document = $(document);

    championships.init();
    $document.find(".race-setup").each(function (index, elem) {
        new RaceSetup($(elem));
    });

    $document.find("#open-in-simres").each(function(index, elem) {
        let link = window.location.href.split("#")[0].replace("results", "results/download") + ".json";

        $(elem).attr('href', "http://simresults.net/remote?result=" + link);

        return false
    });

    serverLogs.init();
    liveTiming.init();
    liveMap.init();


    // init bootstrap-switch
    $.fn.bootstrapSwitch.defaults.size = 'small';
    $.fn.bootstrapSwitch.defaults.animate = false;
    $.fn.bootstrapSwitch.defaults.onColor = "success";
    $document.find("input[type='checkbox']").bootstrapSwitch();

    $document.find('[data-toggle="tooltip"]').tooltip();
    $document.find('[data-toggle="popover"]').popover();

    $document.find(".row-link").click(function () {
        window.location = $(this).data("href");
    });

    $document.find(".results .driver-link").click(function () {
        window.location = $(this).data("href");
        window.scrollBy(0, -100);
    });

    $('form').submit(function () {
        $(this).find('input[type="checkbox"]').each(function () {
            let $checkbox = $(this);
            if ($checkbox.is(':checked')) {
                $checkbox.attr('value', '1');
            } else {
                $checkbox.after().append($checkbox.clone().attr({type: 'hidden', value: 0}));
                $checkbox.prop('disabled', true);
            }
        })
    });

    if ($document.find("form[data-safe-submit]").length > 0) {
        let canSubmit = false;

        $document.find("button[type='submit']").click(function () {
            canSubmit = true;
        });

        // ask the user before they close the webpage
        window.onbeforeunload = function () {
            if (canSubmit) {
                return;
            }

            return "Are you sure you want to navigate away? You'll lose unsaved changes to this setup if you do.";
        };
    }

    $document.find("#CustomRaceScheduled").change(function () {
        if ($(this).val() && $document.find("#CustomRaceScheduledTime").val()) {
            $document.find("#start-race-button").hide();
            $document.find("#save-race-button").val("schedule");
        } else {
            $document.find("#start-race-button").show();
            $document.find("#save-race-button").val("justSave");
        }
    });

    $document.find("#CustomRaceScheduledTime").change(function () {
        if ($(this).val() && $document.find("#CustomRaceScheduled").val()) {
            $document.find("#start-race-button").hide();
            $document.find("#save-race-button").val("schedule");
        } else {
            $document.find("#start-race-button").show();
            $document.find("#save-race-button").val("justSave");

        }
    });
});


const EventCollisionWithCar = 10,
    EventCollisionWithEnv = 11,
    EventNewSession = 50,
    EventNewConnection = 51,
    EventConnectionClosed = 52,
    EventCarUpdate = 53,
    EventCarInfo = 54,
    EventEndSession = 55,
    EventVersion = 56,
    EventChat = 57,
    EventClientLoaded = 58,
    EventSessionInfo = 59,
    EventError = 60,
    EventLapCompleted = 73,
    EventClientEvent = 130,
    EventTrackMapInfo = 222
;


let liveMap = {

    joined: {},

    init: function () {
        const $map = $document.find("#map");

        if (!$map.length) {
            return; // livemap is disabled.
        }

        let ws = new WebSocket(((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host + "/api/live-map");

        let xOffset = 0, zOffset = 0;

        let mapSizeMultiplier = 1;
        let scale = 1;
        let margin = 0;
        let loadedImg = null;
        let mapImageHasLoaded = false;

        let $imgContainer = $map.find("img");

        $(window).resize(function () {
            if (!loadedImg || !mapImageHasLoaded) {
                return;
            }

            mapSizeMultiplier = $imgContainer.width() / loadedImg.width;
        });

        ws.onmessage = function (e) {
            let message = JSON.parse(e.data);

            if (!message) {
                return;
            }

            let data = message.Message;

            switch (message.EventType) {
                case EventVersion:
                    location.reload();
                    break;
                case EventTrackMapInfo:
                    // track map info
                    xOffset = data.OffsetX;
                    zOffset = data.OffsetZ;
                    scale = data.ScaleFactor;
                    break;

                case EventNewConnection:
                    liveMap.joined[data.CarID] = data;

                    let $driverName = $("<span class='name'/>").text(getAbbreviation(data.DriverName));

                    liveMap.joined[data.CarID].dot = $("<div class='dot' style='background: " + randomColor({
                        luminosity: 'bright',
                        seed: data.DriverGUID
                    }) + "'/>").append($driverName);
                    break;

                case EventConnectionClosed:
                    liveMap.joined[data.CarID].dot.remove();
                    delete liveMap.joined[data.CarID];
                    break;

                case EventCarUpdate:
                    liveMap.joined[data.CarID].dot.css({
                        'left': (((data.Pos.X + xOffset + margin)) / scale) * mapSizeMultiplier,
                        'top': (((data.Pos.Z + zOffset + margin)) / scale) * mapSizeMultiplier,
                    });
                    break;

                case EventSessionInfo:
                case EventNewSession:
                    liveMap.clearAllDrivers();
                    let trackURL = "/content/tracks/" + data.Track + (!!data.TrackConfig ? "/" + data.TrackConfig : "") + "/map.png";

                    loadedImg = new Image();

                    loadedImg.onload = function () {
                        $imgContainer.attr({'src': trackURL});

                        if (loadedImg.height / loadedImg.width > 1.2) {
                            // rotate the map
                            $map.addClass("rotated");

                            $imgContainer.css({
                                'max-height': $imgContainer.closest(".map-container").width(),
                                'max-width': 'auto'
                            });

                            mapSizeMultiplier = $imgContainer.width() / loadedImg.width;

                            $map.closest(".map-container").css({
                                'max-height': (loadedImg.width * mapSizeMultiplier) + 20,
                            });


                            $map.css({
                                'max-width': (loadedImg.width * mapSizeMultiplier) + 20,
                            });

                        } else {
                            // un-rotate the map
                            $map.removeClass("rotated");

                            $map.css({
                                'max-height': 'inherit',
                                'max-width': '100%',
                            });

                            $map.closest(".map-container").css({
                                'max-height': 'auto',
                            });

                            $imgContainer.css({
                                'max-height': 'inherit',
                                'max-width': '100%'
                            });

                            mapSizeMultiplier = $imgContainer.width() / loadedImg.width;
                        }

                        mapImageHasLoaded = true;
                    };

                    loadedImg.src = trackURL;
                    break;

                case EventClientLoaded:
                    liveMap.joined[data].dot.appendTo($map);
                    break;

                case EventCollisionWithEnv:
                case EventCollisionWithCar:

                    let x = data.WorldPos.X, y = data.WorldPos.Z;

                    let $collision = $("<div class='collision' />").css({
                        'left': (((x + xOffset + margin)) / scale) * mapSizeMultiplier,
                        'top': (((y + zOffset + margin)) / scale) * mapSizeMultiplier,
                    });

                    $collision.appendTo($map);

                    break;
            }
        };
    },

    clearAllDrivers: function () {
        for (let driver in liveMap.joined) {
            if (driver.dot === undefined) {
                continue;
            }

            driver.dot.delete();
        }

        liveMap.joined = [];
    }
};

function getAbbreviation(name) {
    let parts = name.split(" ");

    if (parts.length < 1) {
        return name
    }

    let lastName = parts[parts.length - 1];

    return lastName.slice(0, 3).toUpperCase();
}


let $entrantTemplate = null;

function makeCarString(cars) {
    let out = "";

    for (let index = 0; index < cars.length; index++) {
        if (index === 0) {
            out = " - " + prettifyName(cars[index], true)
        } else {
            out = out + ", " + prettifyName(cars[index], true)
        }
    }

    return out
}


let liveTiming = {
    init: function () {
        let $liveTimingTable = $document.find("#live-table");

        let raceCompletion = "";
        let total = 0;
        let sessionType = "";
        let lapTime = "";

        if ($liveTimingTable.length) {
            setInterval(function () {
                $.getJSON("/live-timing/get", function (liveTiming) {
                    let date = new Date();

                    // Get lap/laps or time/totalTime
                    if (liveTiming.Time > 0) {
                        total = liveTiming.Time + "m";

                        raceCompletion = timeDiff(liveTiming.SessionStarted, date.getTime());
                    } else if (liveTiming.Laps > 0) {
                        raceCompletion = liveTiming.LapNum;
                        total = liveTiming.Laps + " laps";
                    }

                    let $raceTime = $document.find("#race-time");
                    $raceTime.text("Event Completion: " + raceCompletion + "/ " + total);

                    // Get the session type
                    let $currentSession = $document.find("#current-session");

                    switch (liveTiming.Type) {
                        case 0:
                            sessionType = "Booking";
                            break;
                        case 1:
                            sessionType = "Practice";
                            break;
                        case 2:
                            sessionType = "Qualifying";
                            break;
                        case 3:
                            sessionType = "Race";
                            break;
                    }

                    $currentSession.text("Current Session: " + sessionType);

                    for (let car in liveTiming.Cars) {
                        if (liveTiming.Cars[car].Pos === 0) {
                            liveTiming.Cars[car].Pos = 255
                        }
                    }

                    // Get active cars - sort by pos
                    let sorted = Object.keys(liveTiming.Cars)
                        .sort(function (a, b) {
                            if (liveTiming.Cars[a].Pos < liveTiming.Cars[b].Pos) {
                                return -1
                            } else if (liveTiming.Cars[a].Pos === liveTiming.Cars[b].Pos) {
                                return 0
                            } else if (liveTiming.Cars[a].Pos > liveTiming.Cars[b].Pos) {
                                return 1
                            }
                        });

                    for (let car of sorted) {
                        let $driverRow = $document.find("#" + liveTiming.Cars[car].DriverGUID);
                        let $tr;

                        // Get the lap time, display previous for 10 seconds after completion
                        if (liveTiming.Cars[car].LastLapCompleteTimeUnix + 10000 > date.getTime()) {
                            lapTime = liveTiming.Cars[car].LastLap
                        } else if (liveTiming.Cars[car].LapNum === 0) {
                            lapTime = "0s"
                        } else {
                            lapTime = timeDiff(liveTiming.Cars[car].LastLapCompleteTimeUnix, date.getTime())
                        }

                        if ($driverRow.length) {
                            $driverRow.remove()
                        }

                        $tr = $("<tr/>");
                        $tr.attr({'id': liveTiming.Cars[car].DriverGUID});
                        $tr.empty();

                        let $tdPos = $("<td/>");
                        let $tdName = $("<td/>");
                        let $tdLapTime = $("<td/>");
                        let $tdBestLap = $("<td/>");
                        let $tdGap = $("<td/>");
                        let $tdLapNum = $("<td/>");
                        let $tdEvents = $("<td/>");

                        if (liveTiming.Cars[car].Pos === 255) {
                            $tdPos.text("n/a");
                        } else {
                            $tdPos.text(liveTiming.Cars[car].Pos);
                        }
                        $tr.append($tdPos);

                        $tdName.text(liveTiming.Cars[car].DriverName);
                        $tdName.prepend($("<div class='dot' style='background: " + randomColor({
                            luminosity: 'bright',
                            seed: liveTiming.Cars[car].DriverGUID
                        }) + "'/>"));
                        $tr.append($tdName);

                        $tdLapTime.text(lapTime);
                        $tr.append($tdLapTime);

                        $tdBestLap.text(liveTiming.Cars[car].BestLap);
                        $tr.append($tdBestLap);

                        $tdGap.text(liveTiming.Cars[car].Split);
                        $tr.append($tdGap);

                        $tdLapNum.text(liveTiming.Cars[car].LapNum);
                        $tr.append($tdLapNum);

                        if (liveTiming.Cars[car].Loaded && liveTiming.Cars[car].LoadedTime + 10000 > date.getTime()) {
                            let $tag = $("<span/>");
                            $tag.attr({'class': 'badge badge-success live-badge'});
                            $tag.text("Loaded");

                            $tdEvents.append($tag);
                        }

                        if (liveTiming.Cars[car].Collisions !== null) {
                            for (let y = 0; y < liveTiming.Cars[car].Collisions.length; y++) {
                                if (liveTiming.Cars[car].Collisions[y].Time + 10000 > date.getTime()) {
                                    let $tag = $("<span/>");
                                    $tag.attr({'class': 'badge badge-danger live-badge'});
                                    $tag.text("Crash " + liveTiming.Cars[car].Collisions[y].Type + " at " +
                                        parseFloat(liveTiming.Cars[car].Collisions[y].Speed).toFixed(2) + "m/s");

                                    $tdEvents.append($tag);
                                }
                            }
                        }

                        $tr.append($tdEvents);

                        $liveTimingTable.append($tr)
                    }
                });
            }, 1000);
        }
    }
};


const logCharLimit = 500000;

let serverLogs = {
    init: function () {
        let $serverLog = $document.find("#server-logs");
        let $managerLog = $document.find("#manager-logs");

        let disableServerLogRefresh = false;
        let disableManagerLogRefresh = false;

        $serverLog.on("mousedown", function () {
            disableServerLogRefresh = true;
        });

        $serverLog.on("mouseup", function () {
            disableServerLogRefresh = false;
        });

        $managerLog.on("mousedown", function () {
            disableManagerLogRefresh = true;
        });

        $managerLog.on("mouseup", function () {
            disableManagerLogRefresh = false;
        });

        function isAtBottom($elem) {
            let node = $elem[0];
            return node.scrollTop + node.offsetHeight >= node.scrollHeight - 40;
        }

        if ($serverLog.length && $managerLog.length) {
            setInterval(function () {
                $.get("/api/logs", function (data) {
                    if (!window.getSelection().toString()) {
                        if (data.ServerLog.length > logCharLimit) {
                            data.ServerLog = data.ServerLog.slice(data.ServerLog.length - logCharLimit, data.ServerLog.length);
                        }

                        if (data.ManagerLog.length > logCharLimit) {
                            data.ManagerLog = data.ManagerLog.slice(data.ManagerLog.length - logCharLimit, data.ManagerLog.length);
                        }

                        if (isAtBottom($serverLog) && !disableServerLogRefresh) {

                            $serverLog.text(data.ServerLog);
                            $serverLog.scrollTop(1E10);
                        }

                        if (isAtBottom($managerLog) && !disableManagerLogRefresh) {
                            $managerLog.text(data.ManagerLog);
                            $managerLog.scrollTop(1E10);
                        }
                    }
                });
            }, 1000);
        }
    },
};

function postWithProgressBar(path, data, onSuccess, onFail, $progressBar) {
    $progressBar.closest(".progress").show();
    $progressBar.removeClass("bg-success");

    function showProgress(evt) {
        if (evt.lengthComputable) {
            let percentComplete = Math.round((evt.loaded / evt.total) * 100);
            $progressBar.css('width', percentComplete + '%').attr('aria-valuenow', percentComplete);
            $progressBar.text(percentComplete + "%");

            if (percentComplete === 100) {
                $progressBar.addClass("bg-success");
            }
        }
    }

    $.ajax({
        xhr: function () {
            let xhr = new window.XMLHttpRequest();
            xhr.upload.addEventListener("progress", showProgress, false);
            xhr.addEventListener("progress", showProgress, false);
            return xhr;
        },
        type: 'POST',
        url: path,
        data: data,
        success: onSuccess,
        fail: onFail,
    });
}

const layout = {
    preview: "",
    details: "",
};

let filesToUpload = [];

function submitFiles(path) {
    //JSON encode filestoUpload, JQUERY post request to api endpoint (/api/track/car/upload)
    let newFiles = [];
    let count = 0;

    for (let x = 0; x < filesToUpload.length; x++) {
        // Encode and upload, don't post until all files are read
        let reader = new FileReader();

        reader.readAsDataURL(filesToUpload[x]);

        reader.addEventListener("load", function () {
            newFiles.push({
                'name': filesToUpload[x].name,
                'size': filesToUpload[x].size,
                'type': filesToUpload[x].type,
                'filepath': filesToUpload[x].filepath,
                'dataBase64': reader.result.toString()
            });

            count++;

            if (count === filesToUpload.length) {
                postWithProgressBar(path, JSON.stringify(newFiles), onSuccess, onFail, $("#progress-bar"));
            }
        });
    }
}

function onSuccess(data) {
    console.log("Track Successfully Added");
    location.reload(); // reload for flashes
}

function onFail(data) {
    console.log("Track Could Not be Added");
    location.reload(); // reload for flashes
}

function handleWeatherFiles(fileList) {
    // Correct filepath
    for (let x = 0; x < fileList.length; x++) {
        if (!fileList[x].filepath) {
            fileList[x].filepath = fileList[x].webkitRelativePath;
        }
    }

    // check for multiple weathers inside "weather" folder, if so call loop function for each weather
    if (fileList[0].filepath.startsWith("weather/") && !fileList[0].newPath) {
        let splitList = {};

        for (let y = 0; y < fileList.length; y++) {
            let splitPath = fileList[y].filepath.split("/");

            let weatherIdentifier = splitPath.slice(0, 2).join(":");

            fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

            if (!splitList[weatherIdentifier]) {
                splitList[weatherIdentifier] = []
            }

            splitList[weatherIdentifier].push(fileList[y]);
        }

        for (let weather in splitList) {
            handleWeatherFilesLoop(splitList[weather]);
        }
    } else {
        handleWeatherFilesLoop(fileList);
    }
}

function handleWeatherFilesLoop(fileList) {
    let filesToUploadLocal = [];
    let goodFile = false;

    for (let x = 0; x < fileList.length; x++) {
        // Find the files that the server is interested in
        if (fileList[x].name === "weather.ini" || fileList[x].name.startsWith("preview.")) {
            filesToUploadLocal.push(fileList[x]);

            goodFile = true;
        }
    }

    if (!goodFile) {
        return
    }

    // Preview panel for the weather preset
    let $weatherPanel = $("#weather-info-panel");
    let $row = $("<div/>");
    let $title = $("<h3/>");
    let previewDone = false;

    let weatherName = "";

    if (fileList[0].filepath.startsWith("weather/")) {
        weatherName = fileList[0].filepath.replace('\\', '/').split("/")[1];
    } else {
        weatherName = fileList[0].filepath.replace('\\', '/').split("/")[0];
    }

    $weatherPanel.attr({'class': "card p-3 mt-2"});
    $title.text("Preview: " + weatherName);
    $row.attr({'class': "card-deck"});

    $weatherPanel.append($title);

    $weatherPanel.append($row);

    for (let x = 0; x < filesToUploadLocal.length; x++) {

        // Get a preview image, display livery name
        if (filesToUploadLocal[x].name.startsWith("preview.") && !previewDone) {
            previewDone = true;

            // Set preview to base64 encoded image
            let reader = new FileReader();

            reader.readAsDataURL(filesToUploadLocal[x]);

            reader.addEventListener("load", function () {
                $row.append(buildInfoPanel(reader.result.toString(), "Weather Preview"));
            });
        }
    }

    // Create an upload button that sends queued files to the server
    let $uploadButton = $("#upload-button");
    $uploadButton.attr({'class': "d-inline"});

    if (filesToUploadLocal.length === 0) {
        $uploadButton.text("Sorry, the files you uploaded don't seem to be a compatible weather preset!");
        $uploadButton.empty()
    } else {
        if (!$("#weather-upload-button").length) {
            let $button = $("<button/>");
            $button.attr({
                'class': "btn btn-primary",
                'id': "weather-upload-button"
            });

            $button.click(function (e) {
                e.preventDefault();
                submitFiles("/api/weather/upload")
            });
            $button.text("Upload Weather Preset(s)");

            $uploadButton.append($button);
        }

        for (let x = 0; x < filesToUploadLocal.length; x++) {
            filesToUpload.push(filesToUploadLocal[x])
        }
    }
}

function handleCarFiles(fileList) {
    // Correct filepath
    for (let x = 0; x < fileList.length; x++) {
        if (!fileList[x].filepath) {
            fileList[x].filepath = fileList[x].webkitRelativePath;
        }
    }

    // check for multiple cars inside "cars" folder, if so recall this function for each car
    if (fileList[0].filepath.startsWith("cars/") && !fileList[0].newPath) {
        let splitList = {};

        for (let y = 0; y < fileList.length; y++) {
            let splitPath = fileList[y].filepath.split("/");

            let carIdentifier = splitPath.slice(0, 2).join(":");

            fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

            if (!splitList[carIdentifier]) {
                splitList[carIdentifier] = []
            }

            splitList[carIdentifier].push(fileList[y]);
        }

        for (let car in splitList) {
            handleCarFiles(splitList[car]);
        }
    } else {
        handleCarFilesLoop(fileList);
    }
}

function handleCarFilesLoop(fileList) {
    let filesToUploadLocal = [];
    let goodFile = false;

    for (let x = 0; x < fileList.length; x++) {
        // Find the files that the server is interested in
        if (fileList[x].name === "data.acd" || fileList[x].name === "tyres.ini" || fileList[x].name === "ui_car.json"
            || fileList[x].name.startsWith("livery.") || fileList[x].name.startsWith("preview.")
            || fileList[x].name === "ui_skin.json") {

            filesToUploadLocal.push(fileList[x]);

            if (fileList[x].name === "ui_car.json") {
                goodFile = true;
            }
        }
    }

    if (!goodFile) {
        notA("car");
        return
    }

    let $panel = $("#car-fail");
    $panel.hide();

    // Preview panel for the car
    let $carPanel = $("#car-info-panel");
    let $row = $("<div/>");
    let $title = $("<h3/>");
    let previewDone = false;

    let entrySplit = fileList[0].filepath.replace('\\', '/').split("/");
    let carName = entrySplit[entrySplit.length - 2];

    if (fileList[0].filepath.startsWith("cars/")) {
        carName = fileList[0].filepath.split("/")[1];
    } else {
        carName = fileList[0].filepath.split("/")[0];
    }

    $carPanel.attr({'class': "card p-3 mt-2"});
    $title.text("Preview: " + carName);
    $row.attr({'class': "card-deck"});

    $carPanel.append($title);

    $carPanel.append($row);

    for (let x = 0; x < filesToUploadLocal.length; x++) {

        // Get a preview image, display livery name
        if (filesToUploadLocal[x].name.startsWith("preview.") && !previewDone) {
            previewDone = true;

            let filePathCorrected = filesToUploadLocal[x].filepath.replace('\\', '/');
            let filePathSplit = filePathCorrected.split("/");

            let skinName = filePathSplit[filePathSplit.length - 2];

            // Set preview to base64 encoded image
            let reader = new FileReader();

            reader.readAsDataURL(filesToUploadLocal[x]);

            reader.addEventListener("load", function () {
                $row.append(buildInfoPanel(reader.result.toString(), "Livery: " + skinName));
            });
        }

        // Get info about the car to display in the preview, this often fails due to bad JSON encoding
        if (filesToUploadLocal[x].name === "ui_car.json") {
            let reader = new FileReader();

            reader.readAsText(filesToUploadLocal[x]);

            reader.addEventListener("load", function () {
                let parsed = "";
                let badJSONnoDonut = false;

                try {
                    parsed = JSON.parse(reader.result.toString());
                } catch (error) {
                    badJSONnoDonut = true;
                }

                if (!badJSONnoDonut) {
                    $carPanel.append(buildHtmlTable([parsed]));
                }
            });
        }
    }

    // Create an upload button that sends queued files to the server
    let $uploadButton = $("#upload-button");
    $uploadButton.attr({'class': "d-inline"});

    if (filesToUploadLocal.length === 0) {
        $uploadButton.text("Sorry, the files you uploaded don't seem to be a compatible car!");
        $uploadButton.empty()
    } else {
        if (!$("#car-upload-button").length) {
            let $button = $("<button/>");
            $button.attr({
                'class': "btn btn-primary",
                'id': "car-upload-button"
            });

            $button.click(function (e) {
                e.preventDefault();

                submitFiles("/api/car/upload")
            });
            $button.text("Upload Car(s)");

            $uploadButton.append($button);
        }

        for (let x = 0; x < filesToUploadLocal.length; x++) {
            filesToUpload.push(filesToUploadLocal[x])
        }
    }
}

function getFilesWebkitDataTransferItems(dataTransferItems) {
    function traverseFileTreePromise(item, path = '') {
        return new Promise(resolve => {
            if (item.isFile) {
                item.file(file => {
                    file.filepath = path + file.name; //save full path
                    files.push(file);
                    resolve(file)
                })
            } else if (item.isDirectory) {
                let dirReader = item.createReader();
                dirReader.readEntries(entries => {
                    let entriesPromises = [];

                    for (let entr of entries) {
                        entriesPromises.push(traverseFileTreePromise(entr, path + item.name + "/"));
                    }

                    resolve(Promise.all(entriesPromises))
                })
            }
        })
    }

    let files = [];
    return new Promise((resolve, reject) => {
        let entriesPromises = [];

        for (let it of dataTransferItems) {
            entriesPromises.push(traverseFileTreePromise(it.webkitGetAsEntry()));
        }

        Promise.all(entriesPromises)
            .then(entries => {
                resolve(files)
            })
    })
}

function handleTrackDropFiles(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    dragOutHandler(ev);

    let items = event.dataTransfer.items;
    getFilesWebkitDataTransferItems(items)
        .then(files => {
            handleTrackFiles(files);
        })
}

function handleCarDropFiles(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    dragOutHandler(ev);

    let items = event.dataTransfer.items;
    getFilesWebkitDataTransferItems(items)
        .then(files => {
            handleCarFiles(files);
        })
}

function handleWeatherDropFiles(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    dragOutHandler(ev);

    let items = event.dataTransfer.items;
    getFilesWebkitDataTransferItems(items)
        .then(files => {
            handleWeatherFiles(files);
        })
}

function dragOverHandler(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    document.getElementById("drop-zone").classList.add('drop-zone-hovered');
}

function dragOutHandler(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    document.getElementById("drop-zone").classList.remove('drop-zone-hovered');
}

function handleTrackFiles(fileList) {
    // Correct filepath
    for (let x = 0; x < fileList.length; x++) {
        if (!fileList[x].filepath) {
            fileList[x].filepath = fileList[x].webkitRelativePath;
        }
    }

    if (fileList[0].filepath.startsWith("tracks/") && !fileList[0].newPath) {
        let splitList = {};

        for (let y = 0; y < fileList.length; y++) {
            let splitPath = fileList[y].filepath.split("/");

            let trackIdentifier = splitPath.slice(0, 2).join(":");

            fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

            if (!splitList[trackIdentifier]) {
                splitList[trackIdentifier] = []
            }

            splitList[trackIdentifier].push(fileList[y]);
        }

        for (let track in splitList) {
            handleTrackFilesLoop(splitList[track]);
        }
    } else {
        handleTrackFilesLoop(fileList);
    }
}

function handleTrackFilesLoop(fileList) {
    let layouts = {};
    let layoutNum = 0;
    let filesToUploadLocal = [];
    let trackName = "";
    let goodFile = false;

    for (let x = 0; x < fileList.length; x++) {
        // get model/surfaces and drs zones and ui folder
        if ((fileList[x].name.startsWith("models") && fileList[x].name.endsWith(".ini")) ||
            (fileList[x].name === "surfaces.ini" || fileList[x].name === "drs_zones.ini") ||
            (fileList[x].filepath.includes("/ui/") || fileList[x].name === "map.png" || fileList[x].name === "map.ini")) {

            filesToUploadLocal.push(fileList[x]);
        }

        if (fileList[x].name.startsWith("models")) {
            layoutNum++
        }

        if (fileList[x].name === "surfaces.ini") {
            goodFile = true;
        }
    }

    if (!goodFile) {
        notA("track");
        return
    }

    let $panel = $("#track-fail");
    $panel.hide();

    if (fileList[0].filepath.startsWith("tracks/")) {
        trackName = fileList[0].filepath.split("/")[1];
    } else {
        trackName = fileList[0].filepath.split("/")[0];
    }

    let tableDone = false;
    let $trackPanel = $("#track-info-panel");
    let $row = $("<div/>");
    let $title = $("<h3/>");

    $trackPanel.attr({'class': "card p-3 mt-2"});
    $title.text("Preview: " + trackName);
    $row.attr({'class': "card-deck"});

    $trackPanel.append($title);

    $trackPanel.append($row);

    for (let x = 0; x < filesToUploadLocal.length; x++) {
        if (filesToUploadLocal[x].filepath.includes("/ui/")) {

            if (filesToUploadLocal[x].name === "preview.png") {

                let layoutName = "";

                // For multiple layouts get the layout name and store in map
                if (layoutNum > 1) {
                    let fileListCorrected = filesToUploadLocal[x].filepath.replace('\\', '/');

                    let fileListSplit = fileListCorrected.split("/");

                    layoutName = fileListSplit[fileListSplit.length - 2];
                } else {
                    layoutName = "Default";
                }

                if (!layouts[layoutName]) {
                    layouts[layoutName] = Object.create(layout);
                }

                // Set preview to base64 encoded image
                let reader = new FileReader();

                reader.readAsDataURL(filesToUploadLocal[x]);

                reader.addEventListener("load", function () {
                    layouts[layoutName].preview = reader.result.toString();

                    let layoutInfo = layouts[layoutName];

                    $row.append(buildInfoPanel(layoutInfo.preview, "Layout: " + layoutName));
                });
            }

            if (filesToUploadLocal[x].name === "ui_track.json" && !tableDone) {
                tableDone = true;
                let reader = new FileReader();

                reader.readAsText(filesToUploadLocal[x]);

                reader.addEventListener("load", function () {
                    $trackPanel.append(buildHtmlTable([JSON.parse(reader.result.toString())]));
                });
            }
        }

    }

    let $uploadButton = $("#upload-button");
    $uploadButton.attr({'class': "d-inline"});

    if (filesToUploadLocal.length === 0) {
        $uploadButton.text("Sorry, the files you uploaded don't seem to be a compatible track!");
        $uploadButton.empty()
    } else {
        if (!$("#track-upload-button").length) {
            let $button = $("<button/>");
            $button.attr({
                'class': "btn btn-primary",
                'id': "track-upload-button"
            });
            $button.click(function (e) {
                e.preventDefault();

                submitFiles("/api/track/upload")
            });

            $button.text("Upload Track(s)");

            $uploadButton.append($button);
        }

        for (let x = 0; x < filesToUploadLocal.length; x++) {
            filesToUpload.push(filesToUploadLocal[x])
        }
    }
}

function notA(thing) {
    let $panel = $("#" + thing + "-fail");

    $panel.show();
    $panel.attr({'class': "alert alert-danger mt-2"});
    $panel.text("Sorry, looks like that wasn't a " + thing + "!")
}

function buildInfoPanel(img, info) {
    let $panel = $("<div/>");
    let $img = $("<img/>");
    let $cardBody = $("<div/>");
    let $cardText = $("<h5/>");

    $img.attr({'src': img});
    $img.attr({'alt': "Content Preview"});
    $img.attr({'class': "card-img-top"});

    $cardBody.attr({'class': "card-body"});

    $cardText.attr({'class': "card-title"});
    $cardText.text(info);

    $cardBody.append($cardText);

    $panel.append($img);
    $panel.append($cardBody);

    $panel.attr({'class': "card text-center mb-3"});

    return $panel;
}

// Builds a HTML table from JSON input.
function buildHtmlTable(json) {
    let $cardTable = $("<table/>");

    $cardTable.attr({'class': "table table-sm table-bordered"});
    $cardTable.attr({'id': "layout-table"});

    let columns = addAllColumnHeaders(json, $cardTable);

    for (let i = 0; i < json.length; i++) {
        let $row = $('<tr/>');
        for (let colIndex = 0; colIndex < columns.length; colIndex++) {
            let cellValue = json[i][columns[colIndex]] + "<br>";

            if (cellValue == null) {
                cellValue = "";
            }

            $row.append($('<td/>').html(cellValue));
        }
        $cardTable.append($row);
    }

    return $cardTable
}

// Adds a header row to the table and returns the set of columns.
function addAllColumnHeaders(json, table) {
    let columnSet = [];
    let headerTr$ = $('<tr/>');
    let header$ = $('<thead/>');

    header$.attr({'class': "table-secondary"});

    for (let i = 0; i < json.length; i++) {
        let rowHash = json[i];
        for (let key in rowHash) {
            if ($.inArray(key, columnSet) === -1) {
                if (key === "tags" || key === "run" || key === "url" || key === "torqueCurve" || key === "powerCurve") {
                    continue
                }

                columnSet.push(key);
                headerTr$.append($('<th/>').html(key));
            }
        }
    }

    header$.append(headerTr$);

    table.append(header$);

    return columnSet;
}


let championships = {
    init: function () {
        let $pointsTemplate = $document.find(".points-place").last().clone();

        $document.on("click", ".addEntrant", function (e) {
            e.preventDefault();

            let $raceSetup = $(this).closest(".race-setup");
            let $pointsParent = $raceSetup.find(".points-parent");

            if (!$pointsParent.length) {
                return;
            }

            let $points = $raceSetup.find(".points-place");
            let numEntrants = $raceSetup.find(".entrant:visible").length;
            let numPoints = $points.length;

            for (let i = numPoints; i < numEntrants; i++) {
                // add points up to the numEntrants we have
                let $newPoints = $pointsTemplate.clone();
                $newPoints.find("label").text(ordinalSuffix(i + 1) + " Place");

                let pointsVal = 0;

                // load the default points value for this position
                if (i < defaultPoints.Places.length) {
                    pointsVal = defaultPoints.Places[i];
                }

                $newPoints.find("input").attr({"value": pointsVal});
                $pointsParent.append($newPoints);
            }
        });

        championships.initClassSetup();
    },

    $classTemplate: null,

    initClassSetup: function () {
        let $addClassButton = $document.find("#addClass");
        let $tmpl = $document.find("#class-template");
        championships.$classTemplate = $tmpl.clone();

        $tmpl.remove();

        $addClassButton.click(function (e) {
            e.preventDefault();

            let $cloned = championships.$classTemplate.clone().show();

            $(this).before($cloned);
            new RaceSetup($cloned);
        });

        $document.on("click", ".btn-delete-class", function (e) {
            e.preventDefault();
            $(this).closest(".race-setup").remove();
        });
    },
};

