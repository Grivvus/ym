-- name: ConfirmRestoring :exec
UPDATE public."restore_status" SET
    status = 'finished',
    finished_at = now()
;