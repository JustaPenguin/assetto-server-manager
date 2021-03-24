import {SummernoteWrapper} from "./forms/SummernoteWrapper";

export class Messages {
    public static initSummerNote() {
        let $contentManagerMessageContent = $("#ContentManagerMessageContent");

        if (!$contentManagerMessageContent.length) {
            return;
        }

        let $summerNote = $("#contentManagerWelcomeMessage");
        let html = "";

        if ($contentManagerMessageContent.length > 0) {
            html = $contentManagerMessageContent.html();
        }

        let wrapper = new SummernoteWrapper($summerNote, {
            placeholder: 'A message that Content Manager users can see!',
            tabsize: 2,
            height: 400,
        }, html);

        wrapper.render();
    }
}
