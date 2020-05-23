export class Messages {
    public constructor() {
        this.initSummerNote();
    }

    private initSummerNote(): void {
        let $contentManagerMessageContent = $("#ContentManagerMessageContent");

        if (!$contentManagerMessageContent.length) {
            return;
        }

        let $summerNote = $("#contentManagerWelcomeMessage");

        if ($contentManagerMessageContent.length > 0) {
            $summerNote.summernote('code', $contentManagerMessageContent.html());
        }

        $summerNote.summernote({
            placeholder: 'A message that Content Manager users can see!',
            tabsize: 2,
            height: 400,
        });
    }
}
