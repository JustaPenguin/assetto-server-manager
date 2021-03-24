export class SummernoteWrapper {
    private $element: JQuery;
    private readonly opts: Summernote.Options;
    private readonly code?: string;

    public constructor($element: JQuery, opts: Summernote.Options, code?: string) {
        this.$element = $element;
        this.opts = opts;
        this.code = code;
    }

    public render(): void {
        this.opts.callbacks = {
            onImageUpload: (files: FileList) => {
                for (let file of files) {
                    this.uploadFile(file);
                }
            }
        }

        this.$element.summernote(this.opts);

        if (this.code) {
            this.$element.summernote("code", this.code);
        }
    }

    private uploadFile(file: File) {
        let data = new FormData();
        data.append("image", file);

        $.ajax({
            url: "/api/image-upload",
            type: "POST",
            data: data,
            contentType: false,
            processData: false,
            success: (url: string) => {
                this.$element.summernote("editor.insertImage", url);
            },
        })
    }
}
