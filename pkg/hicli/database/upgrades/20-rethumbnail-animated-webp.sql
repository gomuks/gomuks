-- v20 (compatible with v10+): Mark animated webps for re-thumbnailing
UPDATE media SET thumbnail_error=NULL WHERE thumbnail_error='failed to decode image: webp: invalid format';
