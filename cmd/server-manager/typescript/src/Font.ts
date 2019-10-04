// Font.ts minimises Font-Awesome page load times by only loading in the icons that we are using.
import {library, dom} from "@fortawesome/fontawesome-svg-core";

import {faUser} from "@fortawesome/free-solid-svg-icons/faUser";
import {faBug} from "@fortawesome/free-solid-svg-icons/faBug";
import {faHeart} from "@fortawesome/free-solid-svg-icons/faHeart";
import {faCalendar} from "@fortawesome/free-solid-svg-icons/faCalendar";
import {faBook} from "@fortawesome/free-solid-svg-icons/faBook";
import {faFighterJet} from "@fortawesome/free-solid-svg-icons/faFighterJet";
import {faPalette} from "@fortawesome/free-solid-svg-icons/faPalette";
import {faFlagCheckered} from "@fortawesome/free-solid-svg-icons/faFlagCheckered";
import {faCog} from "@fortawesome/free-solid-svg-icons/faCog";
import {faPollH} from "@fortawesome/free-solid-svg-icons/faPollH";
import {faFileAlt} from "@fortawesome/free-solid-svg-icons/faFileAlt";
import {faRoad} from "@fortawesome/free-solid-svg-icons/faRoad";
import {faCar} from "@fortawesome/free-solid-svg-icons/faCar";
import {faCloudMoon} from "@fortawesome/free-solid-svg-icons/faCloudMoon";
import {faGavel} from "@fortawesome/free-solid-svg-icons/faGavel";
import {faCommentAlt} from "@fortawesome/free-solid-svg-icons/faCommentAlt";
import {faTrash} from "@fortawesome/free-solid-svg-icons/faTrash";
import {faSortUp} from "@fortawesome/free-solid-svg-icons/faSortUp";
import {faSortDown} from "@fortawesome/free-solid-svg-icons/faSortDown";
import {faStar as faSolidStar} from "@fortawesome/free-solid-svg-icons/faStar";
import {faStar as faRegularStar} from "@fortawesome/free-regular-svg-icons/faStar";
import {faFastBackward} from "@fortawesome/free-solid-svg-icons/faFastBackward";
import {faCaretLeft} from "@fortawesome/free-solid-svg-icons/faCaretLeft";
import {faCaretRight} from "@fortawesome/free-solid-svg-icons/faCaretRight";
import {faFastForward} from "@fortawesome/free-solid-svg-icons/faFastForward";
import {faServer} from "@fortawesome/free-solid-svg-icons/faServer";
import {faGithub} from "@fortawesome/free-brands-svg-icons/faGithub";
import {faPlayCircle as faPlayCircleRegular} from "@fortawesome/free-regular-svg-icons/faPlayCircle";
import {faPlayCircle as faPlayCircleSolid} from "@fortawesome/free-solid-svg-icons/faPlayCircle";
import {faTimes} from "@fortawesome/free-solid-svg-icons/faTimes";
import {faChevronLeft} from "@fortawesome/free-solid-svg-icons/faChevronLeft";
import {faChevronRight} from "@fortawesome/free-solid-svg-icons/faChevronRight";
import {faCalendarCheck} from "@fortawesome/free-solid-svg-icons/faCalendarCheck";
import {faClipboardList} from "@fortawesome/free-solid-svg-icons/faClipboardList";
import {faUsersCog} from "@fortawesome/free-solid-svg-icons/faUsersCog";
import {faArrowRight} from "@fortawesome/free-solid-svg-icons/faArrowRight";
import {faArrowDown} from "@fortawesome/free-solid-svg-icons/faArrowDown";
import {faPencilAlt} from "@fortawesome/free-solid-svg-icons/faPencilAlt";
import {faCarCrash} from "@fortawesome/free-solid-svg-icons/faCarCrash";

library.add(
    faUser,
    faBug,
    faHeart,
    faCalendar,
    faBook,
    faFighterJet,
    faPalette,
    faFlagCheckered,
    faCog,
    faPollH,
    faFileAlt,
    faRoad,
    faCar,
    faCloudMoon,
    faGavel,
    faCommentAlt,
    faTrash,
    faSortUp,
    faSortDown,
    faRegularStar,
    faSolidStar,
    faPlayCircleRegular,
    faPlayCircleSolid,
    faFastBackward,
    faCaretLeft,
    faCaretRight,
    faFastForward,
    faServer,
    faGithub,
    faTimes,
    faChevronLeft,
    faChevronRight,
    faCalendarCheck,
    faClipboardList,
    faUsersCog,
    faArrowRight,
    faArrowDown,
    faPencilAlt,
    faCarCrash,
);

dom.watch();