import "jquery";
import "bootstrap";
import "bootstrap-switch";
import "summernote/dist/summernote-bs4";
import "./libs/jquery.quicksearch.js";
import "multiselect";
import "moment";
import "moment-timezone";
import "./javascript/manager.js";
import { RaceControl } from "./RaceControl";
import "./Font";
import {CarDetail} from "./CarDetail";
import "./Calendar";

$(() => {
    new RaceControl();
    new CarDetail();
});

declare global {
   interface JQuery {
      bootstrapSwitch: any;
   }
}
