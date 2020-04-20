export class HostedIntroPopup {
    public constructor() {
        const $introModal = $("#intro-modal");

        $introModal.on('shown.bs.modal', () => {
            $.get("/accounts/dismiss-intro");
        });

        $introModal.modal();
    }
}