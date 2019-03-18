export class ServerLogs {
    private readonly $serverLog: JQuery<HTMLElement>;
    private readonly $managerLog: JQuery<HTMLElement>;

    private disableServerLogRefresh = false;
    private disableManagerLogRefresh = false;

    public constructor() {
        this.$serverLog = $("#server-logs");
        this.$managerLog = $("#manager-logs");

        if (!this.$serverLog.length || !this.$managerLog.length) {
            return;
        }

        this.$serverLog.on("mousedown", () => {
            this.disableServerLogRefresh = true;
        });

        this.$serverLog.on("mouseup", () => {
            this.disableServerLogRefresh = false;
        });

        this.$managerLog.on("mousedown", () => {
            this.disableManagerLogRefresh = true;
        });

        this.$managerLog.on("mouseup", () => {
            this.disableManagerLogRefresh = false;
        });

        setInterval(() => {
            $.get("/api/logs", (data) => {
                if (!window.getSelection().toString()) {
                    if (this.isAtBottom(this.$serverLog) && !this.disableServerLogRefresh) {
                        this.$serverLog.text(data.ServerLog);
                        this.$serverLog.scrollTop(1E10);
                    }

                    if (this.isAtBottom(this.$managerLog) && !this.disableManagerLogRefresh) {
                        this.$managerLog.text(data.ManagerLog);
                        this.$managerLog.scrollTop(1E10);
                    }
                }
            });
        }, 1000);
    }

    private isAtBottom($elem: JQuery<HTMLElement>) {
        let node = $elem[0];
        return node.scrollTop + node.offsetHeight >= node.scrollHeight - 40;
    }
}