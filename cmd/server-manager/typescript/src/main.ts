import "jquery";
import "bootstrap";
import "bootstrap-switch";
import "summernote/dist/summernote-bs4";
import "multiselect";
import "moment";
import "moment-timezone";
import "./javascript/manager.js";
import "./Font";
import "./Calendar";
import "./RaceList";

import {RaceControl} from "./RaceControl";
import {CarDetail} from "./CarDetail";
import {TrackDetail} from "./TrackDetail";
import {CarList} from "./CarList";
import {RaceWeekend} from "./RaceWeekend";
import {ChangelogPopup} from "./ChangelogPopup";
import {Messages} from "./Messages";
import {Championship} from "./Championship";
import {Results} from "./Results";
import {RaceSetup} from "./RaceSetup";

$(() => {
    new RaceControl();
    new CarDetail();
    new TrackDetail();
    new CarList();
    new RaceWeekend.View();
    new RaceWeekend.EditSession();
    new ChangelogPopup();
    new Messages();

    new Championship.Edit();
    new Championship.SignUpForm();
    new Championship.View();

    new Results();

    $(".race-setup").each(function (index, elem) {
        new RaceSetup($(elem));
    });
});

declare global {
    interface JQuery {
        multiSelect: any;
    }
}