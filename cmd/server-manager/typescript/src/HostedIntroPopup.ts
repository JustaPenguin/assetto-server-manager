export class HostedIntroPopup {
    public constructor() {
        const $introModal = $("#intro-modal");

        if (!$introModal.length) {
            return;
        }

        $introModal.on('shown.bs.modal', () => {
            $.get("/accounts/dismiss-intro");
        });

        $introModal.modal();
    }
}
