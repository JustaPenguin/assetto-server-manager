import "jquery";
import "bootstrap";
import "bootstrap-switch";
import "summernote/dist/summernote-bs4";
import "multiselect";
import "moment";
import "moment-timezone";
import {EntryPoint} from "./javascript/manager";
import "./Font";
import "./Calendar";

import {RaceControl} from "./RaceControl";
import {CarDetail} from "./CarDetail";
import {TrackDetail} from "./TrackDetail";
import {CarSearch} from "./CarSearch";
import {CarList} from "./CarList";
import {RaceWeekend} from "./RaceWeekend";
import {ChangelogPopup} from "./ChangelogPopup";
import {HostedIntroPopup} from "./HostedIntroPopup";
import {Messages} from "./Messages";
import {Championship} from "./Championship";
import {Results} from "./Results";
import {RaceList} from "./RaceList";
import {SpectatorCar} from "./SpectatorCar";
import {Form} from "./Form";

$(() => {
    new Form();
    EntryPoint();
    new RaceControl();
    new CarDetail();
    new TrackDetail();
    new CarList();
    new RaceWeekend.View();
    new RaceWeekend.EditSession();
    new ChangelogPopup();
    new HostedIntroPopup();
    Messages.initSummerNote();
    new Championship.View();
    new Results();
    new RaceList();
    new SpectatorCar();

    $(".race-setup").each(function (index, elem) {
        new CarSearch($(elem));
    });
});

declare global {
    interface JQuery {
        multiSelect: any;
        select2: any;
    }
}
