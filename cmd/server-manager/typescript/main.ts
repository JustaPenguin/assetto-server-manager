import "jquery";
import "bootstrap";
import "bootstrap-switch";
import "@fortawesome/fontawesome-free/js/all";
import "summernote";
import "multiselect";
import "./libs/jquery.quicksearch.js";
import "moment";
import "moment-timezone";
import { RaceControl } from "./RaceControl";
import "./javascript/manager.js";

$(() => {
    new RaceControl();
});

declare global {
   interface JQuery {
      bootstrapSwitch: any;
   }
}
