-- Script para limpiar gimnasio, horarios, fotos y archivos asociados
DO $$
DECLARE
    target_gym_id BIGINT := 2; -- <-- COLOCA AQUÍ EL ID DE TU GIMNASIO (o 0 para limpiar todos)
    photo_file_ids BIGINT[];
    bg_file_id BIGINT;
BEGIN
    IF target_gym_id > 0 THEN
        -- 1. Capturar IDs de archivos en variables antes de borrar las relaciones
        SELECT background_id INTO bg_file_id FROM gyms WHERE id = target_gym_id;
        SELECT ARRAY_AGG(file_id) INTO photo_file_ids FROM gym_photos WHERE gym_id = target_gym_id;

        -- 2. Desvincular portada en gyms (rompe FK: gyms.background_id -> files.id)
        UPDATE gyms SET background_id = NULL WHERE id = target_gym_id;

        -- 3. Eliminar gym_photos PRIMERO (rompe FK: gym_photos.file_id -> files.id)
        DELETE FROM gym_photos WHERE gym_id = target_gym_id;

        -- 4. Eliminar horarios y encargados
        DELETE FROM gym_schedules WHERE gym_id = target_gym_id;
        DELETE FROM gym_managers WHERE gym_id = target_gym_id;

        -- 5. Eliminar el gimnasio
        DELETE FROM gyms WHERE id = target_gym_id;

        -- 6. Ahora que nada referencia a 'files', eliminar los registros de files
        IF photo_file_ids IS NOT NULL THEN
            DELETE FROM files WHERE id = ANY(photo_file_ids);
        END IF;

        IF bg_file_id IS NOT NULL THEN
            DELETE FROM files WHERE id = bg_file_id;
        END IF;

        RAISE NOTICE 'Gimnasio % y todos sus datos relacionados fueron eliminados exitosamente.', target_gym_id;
    ELSE
        -- Limpieza total de todos los gimnasios si target_gym_id <= 0
        SELECT ARRAY_AGG(file_id) INTO photo_file_ids FROM gym_photos;

        UPDATE gyms SET background_id = NULL;
        DELETE FROM gym_photos;
        DELETE FROM gym_schedules;
        DELETE FROM gym_managers;
        DELETE FROM gyms;

        IF photo_file_ids IS NOT NULL THEN
            DELETE FROM files WHERE id = ANY(photo_file_ids);
        END IF;

        RAISE NOTICE 'Todos los gimnasios y fotos fueron eliminados exitosamente.';
    END IF;
END $$;

