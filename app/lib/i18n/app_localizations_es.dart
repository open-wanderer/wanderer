// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Spanish Castilian (`es`).
class AppLocalizationsEs extends AppLocalizations {
  AppLocalizationsEs([String locale = 'es']) : super(locale);

  @override
  String get biking => 'Biking';

  @override
  String get canoeing => 'Canoeing';

  @override
  String get walking => 'Walking';

  @override
  String get about => 'Sobre';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Estás al punto de borrar tu cuenta. Todas tus rutas también serán borradas. ¿Quieres proceder?';

  @override
  String get account_privacy => 'Privacidad de la cuenta';

  @override
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Actividades',
      one: 'Actividad',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => 'Añadir biografía';

  @override
  String get add_entry => 'Añadir entrada';

  @override
  String get add_to_list => 'Añadir a la lista';

  @override
  String get add_waypoint => 'Añadir Punto de Interés';

  @override
  String get added_trail_to => 'Ruta añadida a';

  @override
  String get added_trails_to => 'Rutas añadidas a';

  @override
  String get after => 'Después';

  @override
  String get all => 'All';

  @override
  String get all_activities => 'Todas las actividades';

  @override
  String get allow_auto_geolocate =>
      'Begin drawing a new trail from your current location';

  @override
  String get alphabetical => 'Alfabético';

  @override
  String get already_account => '¿Ya tienes una cuenta?';

  @override
  String get altitude => 'Altitud';

  @override
  String get amenity => 'Amenity';

  @override
  String get api_documentation => 'Documentación API';

  @override
  String get api_tokens => 'API Tokens';

  @override
  String get api_tokens_hint =>
      'API Tokens can be used to grant 3rd party applications access to your wanderer account.';

  @override
  String get apply_user_settings => 'Apply user settings';

  @override
  String get attraction => 'Atracción';

  @override
  String get author => 'Autor';

  @override
  String get avatar => 'Avatar';

  @override
  String get average_speed => 'Velocidad media';

  @override
  String get avoid_bad_surfaces => 'Evitar superficies malas';

  @override
  String get back => 'Atrás';

  @override
  String get back_to_login => 'Atrás al acceso';

  @override
  String get bakery => 'Pastelería';

  @override
  String get barrier => 'Barrera';

  @override
  String get basic_info => 'Información básica';

  @override
  String get basque => 'Vasco';

  @override
  String get before => 'Antes';

  @override
  String get behavior => 'Behavior';

  @override
  String get bicycle_parking => 'Aparcamiento de bicicletas';

  @override
  String get bicycle_rental => 'Alquiler de bicicletas';

  @override
  String get bicycle_shop => 'Tienda de bicicletas';

  @override
  String get bike_type => 'Tipo de bici';

  @override
  String get bus_stop => 'Parada de autobús';

  @override
  String get by => 'de';

  @override
  String get campsite => 'Campamento';

  @override
  String get can => 'puede';

  @override
  String get cancel => 'Borrar';

  @override
  String get car => 'Coche';

  @override
  String get car_motorcycle => 'Coche/Moto';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Fichas',
      one: 'Ficha',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Categorías';

  @override
  String get category => 'Categoría';

  @override
  String get change => 'Modificar';

  @override
  String get change_email => 'Cambiar mail';

  @override
  String get change_password => 'Cambiar password';

  @override
  String get changelog => 'Registro de cambios';

  @override
  String get chinese => 'Chinés (simplificado)';

  @override
  String get clear_all => 'Limpiar todo';

  @override
  String get climbing => 'Escalada';

  @override
  String get close => 'Cerrar';

  @override
  String get collapse_trail_list => 'Collapse trail list';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Comentarios',
      one: 'Comentario',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Completado';

  @override
  String get completed_a_trail => 'completó una ruta';

  @override
  String get completed_tours => 'Tours completados';

  @override
  String get completion_status => 'Estado de avance';

  @override
  String get confirm => 'Confirmar';

  @override
  String get confirm_deletion => 'Confirmar Borrado';

  @override
  String get confirm_publish => 'Confirmar publicación';

  @override
  String get confirm_share => 'Confirmar compartir';

  @override
  String get connect => 'Conectar';

  @override
  String get contribute => 'Contribuir';

  @override
  String get couldnt_start_navigation =>
      'Couldn\'t start navigation. Check your connection and try again.';

  @override
  String get location_services_disabled =>
      'Location services are disabled. Please enable GPS to use navigation.';

  @override
  String get location_permission_denied =>
      'Location permission is required for navigation.';

  @override
  String get location_permission_permanently_denied =>
      'Location permission is permanently denied. Please enable it in Settings.';

  @override
  String get copy_link => 'Copiar Enlace';

  @override
  String get create_new_list => 'Crear una nueva lista';

  @override
  String get create_waypoint => 'Crear punto de ruta';

  @override
  String get creation_date => 'Fecha de creación';

  @override
  String get crop => 'Recortar';

  @override
  String get cross => 'Cruzar';

  @override
  String get current_password => 'Contraseña actual';

  @override
  String get cycling => 'Ciclismo';

  @override
  String get cycling_speed => 'Velocidad de ciclismo';

  @override
  String get czech => 'Czech';

  @override
  String get danger_zone => 'Zona peligrosa';

  @override
  String get date => 'Fecha';

  @override
  String get default_category => 'Categoría predefinida';

  @override
  String get default_location => 'Ubicación predefinida';

  @override
  String get degrees => 'Grados';

  @override
  String get delete => 'Borrar';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Borrar Cuenta';

  @override
  String get delete_list_confirm =>
      '¿Seguro quieres borrar esta lista? Las rutas en la lista seguirán estando disponibles.';

  @override
  String get delete_summit_log_confirm =>
      '¿Seguro que quieres borrar esta ruta? Esta acción no puede restablecerse.';

  @override
  String get delete_trail_confirm =>
      '¿Seguro que quieres borrar esta ruta? Esta acción no puede restablecerse.';

  @override
  String get describe_your_trail => 'Describe tu ruta';

  @override
  String get description => 'Descripción';

  @override
  String get difficult => 'Dificultad';

  @override
  String get difficulty => 'Dificultad';

  @override
  String get directions => 'Indicaciones';

  @override
  String get display => 'Ver';

  @override
  String get display_as => 'Mostrar como';

  @override
  String get distance => 'Distancia';

  @override
  String get documentation => 'Documentación';

  @override
  String get download => 'Descarga';

  @override
  String get draw_a_route => 'Dibujar una ruta';

  @override
  String get driving => 'Conducir';

  @override
  String get duplicate => 'Duplicar';

  @override
  String get duration => 'Duración';

  @override
  String get dutch => 'Holandés';

  @override
  String get easy => 'Fácil';

  @override
  String get edit => 'Editar';

  @override
  String get edit_entry => 'Editar entrada';

  @override
  String get edit_list => 'Editar Lista';

  @override
  String get edit_route => 'Editar ruta';

  @override
  String get edit_waypoint => 'Editar Punto de Interés';

  @override
  String get edited => 'editado';

  @override
  String get elevation_gain => 'Ascenso';

  @override
  String get elevation_loss => 'Descenso';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => 'Correo electrónico';

  @override
  String get email_not_unique => 'This email address is already in use.';

  @override
  String get email_updated => 'Correo electrónico actualizado';

  @override
  String get email_verified => 'Correo verificado';

  @override
  String empty_activities(Object username) {
    return '$username aún no tiene actividad';
  }

  @override
  String empty_bio(Object username) {
    return '$username aún no ha añadido una biografía';
  }

  @override
  String get empty_feed => 'Tus noticias estan vacías';

  @override
  String get empty_feed_explanation =>
      'Tus actividades o de las personas que sigas, aparecerán aquí';

  @override
  String empty_lists(Object username) {
    return '$username no tiene listas públicas';
  }

  @override
  String get enable_auto_routing => 'Habilitar auto-ruta';

  @override
  String get english => 'Inglés';

  @override
  String get entry => 'Entrada';

  @override
  String get error_copying_trail => 'Error copying trail';

  @override
  String get error_creating_user => 'Error creando el usuario';

  @override
  String get error_deleting_token => 'Error deleting token';

  @override
  String get error_disabling_strava_integration =>
      'Error al desactivar la integración de strava';

  @override
  String get error_during_login => 'Error durante el acceso';

  @override
  String get error_during_password_reset =>
      'Imposible enviar la contraseña de restablecimiento al correo electrónico';

  @override
  String get error_exporting_trail => 'Error exportando la ruta';

  @override
  String get error_generating_token => 'Error generating token';

  @override
  String get error_liking_trail => 'Error al dar \"me gusta\" a la ruta';

  @override
  String get error_logging_in_to_hammerhead => 'Error logging in to Hammerhead';

  @override
  String get error_logging_in_to_komoot => 'Error al iniciar sesión en komoot';

  @override
  String get error_posting_comment => 'Error publicando el comentario';

  @override
  String get error_printing_map => 'Error durante la impresión del mapa';

  @override
  String get error_reading_file => 'Error leyendo el archivo';

  @override
  String get error_saving_list => 'Error guardando la lista';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Error guardando la ruta';

  @override
  String error_setting_up_integration(Object provider) {
    return 'Error al configurar la integración con $provider';
  }

  @override
  String get error_updating_hammerhead_integration =>
      'Error updating Hammerhead integration';

  @override
  String get error_updating_komoot_integration =>
      'Error updating komoot integration';

  @override
  String get error_updating_password => 'Error actualizando la contraseña';

  @override
  String get error_updating_strava_integration =>
      'Error al actualizar la integración con komoot';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Error uploading trail to Hammerhead';

  @override
  String get est_duration => 'Duración estimada';

  @override
  String get everyone_with_the_link => 'Cualquier persona con el enlace';

  @override
  String get expand_trail_list => 'Expand trail list';

  @override
  String get expiration => 'Expiration';

  @override
  String get expires => 'Expires';

  @override
  String get explore => 'Explora';

  @override
  String get explore_some_trails => 'Explora alguna ruta';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get export => 'Exportar';

  @override
  String get export_all_trails => 'Exportar todas las rutas';

  @override
  String get favourite_sport => 'Deporte favorito';

  @override
  String get features => 'Funcionalidades';

  @override
  String get ferry => 'Ferri';

  @override
  String get file_format => 'Formato del archivo';

  @override
  String file_too_big(Object file, Object size) {
    return 'El archivo $file es demasiado grande (máx. $size)';
  }

  @override
  String get filter_categories => 'Filtrar categorías';

  @override
  String get filter_difficulty => 'Filtrar dificultad';

  @override
  String get filter_tags => 'Filtrar etiquetas';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => 'Finalizar';

  @override
  String get finish_disabled_hint =>
      'Add at least 2 anchors to finish your route.';

  @override
  String get fixed_speed => 'Velocidad fijada';

  @override
  String get focus_map_on => 'Centrar mapa sobre';

  @override
  String get follow => 'Seguir';

  @override
  String get follow_request_pending => 'Solicitud pendiente';

  @override
  String get followers => 'Seguidores';

  @override
  String get following => 'Siguiendo';

  @override
  String get food => 'Comida';

  @override
  String get food_drinks => 'Comida y bebida';

  @override
  String get forgot_your_password => '¿Contraseña olvidada?';

  @override
  String get french => 'Francés';

  @override
  String get from_file => 'Desde archivo';

  @override
  String get from_photos => 'Desde fotos';

  @override
  String get from_url => 'Desde URL';

  @override
  String get garage => 'Garaje';

  @override
  String get gas_station => 'Gasolinera';

  @override
  String get generate_new_token => 'Generate new token';

  @override
  String get german => 'Alemán';

  @override
  String get get_position_from_exif =>
      'Obtener las coordenadas de los datos EXIF';

  @override
  String get get_started => 'Iniciar';

  @override
  String get grid => 'Cuadricula';

  @override
  String get grocery_store => 'Supermercado';

  @override
  String get hammerhead_integration_after_date_hint =>
      'If your hammerhead account is already synced with other trail databases, such as komoot or Strava, start syncing your Hammerhead data may result in duplicates. To avoid this, you can set an start date below, meaning only activities recorded after this date will be synced.';

  @override
  String get heading => 'Título';

  @override
  String get height => 'Altura';

  @override
  String get help => 'Ayuda';

  @override
  String get hero_section_0_text =>
      'Explore exciting trails, save your favorites, and experience the beauty of nature. Find your next adventure!';

  @override
  String get hero_section_1_heading => 'It seems there are no trails here yet.';

  @override
  String get hero_section_1_text =>
      'Here are some trails you might like. Or you can just go to the full list right now.';

  @override
  String get hero_section_1_text_alternative =>
      'Get started by saving your latest adventure.';

  @override
  String get hero_section_2_text =>
      'Did you know? You cannot only save you hiking trails. There are many categories for all your adventures.';

  @override
  String get hiking => 'Senderismo';

  @override
  String get home => 'Inicio';

  @override
  String get hotel => 'Hotel';

  @override
  String get hungarian => 'Húngaro';

  @override
  String get hut => 'Cabaña';

  @override
  String get hybrid => 'Hibrido';

  @override
  String get icon => 'Icono';

  @override
  String get ignore_trails_before_date => 'Ignore trails before this date';

  @override
  String get imperial => 'Imperial';

  @override
  String get import => 'Importar';

  @override
  String get import_hint =>
      'Selecciona o arrastra aquí archivos GPX, FIT, KML o TCX...';

  @override
  String get include_description => 'Incluir descripción';

  @override
  String get include_waypoints => 'Incluir puntos de interés';

  @override
  String get integration_description_hammerhead =>
      'Syncs your Hammerhead tours with wanderer in regular intervals.';

  @override
  String get integration_description_komoot =>
      'Sincroniza tus recorridos de Komoot con Wanderer en intervalos regulares.';

  @override
  String get integration_description_strava =>
      'Sincroniza tus recorridos y actividades de Strava con Wanderer en intervalos regulares.';

  @override
  String get integration_disabled => 'integración desactivada';

  @override
  String get integration_enabled => 'integración activada';

  @override
  String get integration_privacy_hint_original =>
      'Imported trails will maintain the same visibility they have on the external platform. For example, if the original trail was public, it will be public in wanderer, even if trails are private by default according to your privacy settings.';

  @override
  String get integration_privacy_hint_user =>
      'The original trail\'s visibility is discarded. Instead, the local privacy settings for trails are applied to all imported trails.';

  @override
  String get integrations => 'Integraciones';

  @override
  String get invalid_date => 'Fecha no válida';

  @override
  String get invalid_username => 'Usuario no válido';

  @override
  String get italian => 'Italiano';

  @override
  String get joined => 'Afiliado';

  @override
  String get keep_original => 'Keep original';

  @override
  String get keep_private => 'Keep private';

  @override
  String get language => 'Idioma';

  @override
  String get last_used => 'Last used';

  @override
  String get latitude => 'Latitud';

  @override
  String layer(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Listas',
      one: 'Lista',
    );
    return '$_temp0';
  }

  @override
  String get license => 'Licencia';

  @override
  String get like_status => 'Estado \"Me gusta\"';

  @override
  String get liked => 'Me gusta';

  @override
  String get likes => 'Me gusta';

  @override
  String get limited => 'Limitado';

  @override
  String get link_copied => '¡Enlace copiado!';

  @override
  String get linked_lists => 'Linked lists';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Listas',
      one: 'Lista',
    );
    return '$_temp0';
  }

  @override
  String get list_not_shared => 'No compartido con ninguno';

  @override
  String get list_public_warning =>
      'Todas las rutas en esta lista serán públicas.';

  @override
  String get list_saved_successfully => 'Lista guardada con éxito';

  @override
  String get list_share_warning =>
      'Compartir una lista automáticamente comparte todas las rutas que contiene.';

  @override
  String get list_share_warning_update =>
      'Las rutas añadidas serán compartidas con todos los usuarios que tienen acceso a esta lista.';

  @override
  String get location => 'Ubicación';

  @override
  String get locations => 'Locations';

  @override
  String get location_tracking_notification_title => 'Wanderer';

  @override
  String get location_tracking_notification_text => 'Recording your trail';

  @override
  String get login => 'Acceso';

  @override
  String get login_details => 'Detalles de acceso';

  @override
  String get logout => 'Salir';

  @override
  String get longitude => 'Longitud';

  @override
  String get loop => 'Bucle';

  @override
  String get make_one => '¡Crea uno!';

  @override
  String get make_thumbnail => 'Generar miniaturas';

  @override
  String get map => 'Mapa';

  @override
  String get map_style => 'Estilo de mapa';

  @override
  String get max_hiking_difficulty => 'Dificultad máxima';

  @override
  String get metric => 'Métrica';

  @override
  String get moderate => 'Medio';

  @override
  String get more => 'Más';

  @override
  String get more_route_settings => 'Más opciones de ruta';

  @override
  String get mountain => 'Montaña';

  @override
  String get mountain_pass => 'Paso de montaña';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return 'Tiene que tener por lo menos $n caracteres';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return 'Tiene que tener por lo menos $n caracteres';
  }

  @override
  String get my_account => 'Mi cuenta';

  @override
  String get my_trails => 'Mis rutas';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return 'hace $n días';
  }

  @override
  String n_hours_ago(Object n) {
    return 'hace $n horas';
  }

  @override
  String n_minutes_ago(Object n) {
    return 'hace $n minutos';
  }

  @override
  String n_months_ago(Object n) {
    return 'hace $n meses';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'hace $n segundos',
      zero: 'ahora',
      one: '',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return 'hace $n años';
  }

  @override
  String get name => 'Nombre';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => 'Cerca';

  @override
  String get never => 'Never';

  @override
  String get new_list => 'Nueva lista';

  @override
  String get new_password => 'Nueva contraseña';

  @override
  String get new_password_error => 'Error estableciendo la nueva contraseña';

  @override
  String get new_password_success => 'La nueva contraseña ha sido configurada';

  @override
  String get new_password_text => 'Configura una nueva contraseña';

  @override
  String get new_token_generated => 'New API Token generated';

  @override
  String get new_trail => 'Nueva Ruta';

  @override
  String get trail_source_planner => 'Open trail planner';

  @override
  String get trail_source_record => 'Record trail';

  @override
  String get trail_source_import => 'Import file';

  @override
  String get trail_source_import_error => 'Could not import file';

  @override
  String get travel_profile_hike => 'Hike';

  @override
  String get travel_profile_hike_description =>
      'Plan a route for walking or hiking.';

  @override
  String get travel_profile_bike => 'Bike';

  @override
  String get travel_profile_bike_description => 'Plan a route for cycling.';

  @override
  String get coming_soon => 'Coming soon';

  @override
  String get no_account => '¿No tienes una cuenta?';

  @override
  String get no_api_tokens => 'You have no API Tokens';

  @override
  String get no_comments_so_far => 'Ningún comentario todavía';

  @override
  String get no_data => 'No datos';

  @override
  String get no_description_for_now => 'Ninguna descripción de momento';

  @override
  String get no_gps_data_in_image => 'Sin datos GPS en la imagen';

  @override
  String get no_grid => 'Ninguna cuadrícula';

  @override
  String get no_notifications => 'No notificaciones';

  @override
  String get no_photos_here => 'No fotos aquí';

  @override
  String get no_preference => 'Ninguna preferencia';

  @override
  String get no_results => 'Ningún resultado encontrado';

  @override
  String get no_routes_added => 'Ninguna ruta añadida';

  @override
  String get no_waypoints_yet => 'No Puntos de Interés todavía';

  @override
  String get norwegian => 'Norwegian';

  @override
  String get not_a_valid_email_address => 'Correo electrónico no válido';

  @override
  String get not_a_valid_url => 'No es un URL válida';

  @override
  String get not_completed => 'No completado';

  @override
  String notification_comment_mention(Object user) {
    return '$user te ha mencionado en un comentario';
  }

  @override
  String notification_list_create(Object user) {
    return '$user ha creado una nueva lista';
  }

  @override
  String notification_list_share(Object user) {
    return '$user ha compartido una nueva lista contigo';
  }

  @override
  String get notification_new_follower => 'Tienes un nuevo seguidor';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user ha dejado un comentario sobre tu ruta \"$trail\"';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user te ha mencionado en un comentario';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user ha dejado un comentario sobre tu ruta \"$trail\"';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user ha creado una nueva ruta';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user le ha gustado tu ruta \"$trail\"';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user te ha mencionado en una ruta';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user ha compartido una ruta contigo';
  }

  @override
  String get notifications => 'Notificaciones';

  @override
  String object_share_error(Object object) {
    return 'Un $object debe ser público para ser compartido entre instancias.';
  }

  @override
  String get off => 'Apagado';

  @override
  String get only_me => 'Solo yo';

  @override
  String get open_in_new_tab => 'Abrir en nueva pestaña';

  @override
  String get or => 'o';

  @override
  String get orientation => 'Orientación';

  @override
  String get paper_size => 'Tamaño del papel';

  @override
  String get paragraph => 'Párrafo';

  @override
  String get parking => 'Aparcamiento';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Contraseña';

  @override
  String get password_confirm => 'Confirmar contraseña';

  @override
  String get password_reset_sent =>
      'Correo electrónico para restablecer la contraseña enviado';

  @override
  String get password_reset_text =>
      'Enviaremos un enlace para restablecer la contraseña a tu correo electrónico.';

  @override
  String get password_updated => 'Contraseña actualizada';

  @override
  String get passwords_must_match => 'Las contraseñas tienen que coincidir';

  @override
  String get photos => 'Fotos';

  @override
  String photos_skipped_no_gps(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count photos skipped — no GPS data',
      one: '1 photo skipped — no GPS data',
    );
    return '$_temp0';
  }

  @override
  String get pick_a_trail => 'Escoge una ruta';

  @override
  String get planned_a_trail => 'planificada una ruta';

  @override
  String get planned_tours => 'Visitas planificadas';

  @override
  String get pois => 'Puntos de interés';

  @override
  String get polish => 'Polaco';

  @override
  String get portuguese => 'Portugués';

  @override
  String get print => 'Imprimir';

  @override
  String get privacy => 'Privacidad';

  @override
  String get private => 'Privado';

  @override
  String get profile => 'Perfil';

  @override
  String get public => 'Público';

  @override
  String get public_access => 'Acceso público';

  @override
  String get public_share_everyone =>
      'Todos en Internet con el enlace pueden ver esta ruta';

  @override
  String get public_share_limited =>
      'Sólo las personas con acceso pueden abrir el enlace';

  @override
  String get public_transport => 'Transporte público';

  @override
  String get radius => 'Radio';

  @override
  String get railway_station => 'Estación de tren';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get read_more => 'Leer más';

  @override
  String get ready_to_join => 'Listo para unirse';

  @override
  String get recalculate_elevation_data => 'Recalcular datos de elevación';

  @override
  String get recalculating_elevation_data_hint =>
      'Recalcular los datos de elevación borrará los datos de elevación existentes, si los hay, y los reemplazará con los datos de Valhalla.';

  @override
  String get register => 'Registrar';

  @override
  String get remote_users_cannot_edit => 'Usuarios remotos no pueden editar';

  @override
  String get removed_trail_from => 'Ruta borrada de';

  @override
  String get removed_trails_from => 'Rutas borrada de';

  @override
  String get required => 'Obligatorio';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Restablecer';

  @override
  String get reset_password => 'Restablecer Contraseña';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get reverse_direction => 'Invertir la dirección';

  @override
  String get road => 'Carretera';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Rutas',
      one: 'Ruta',
    );
    return '$_temp0';
  }

  @override
  String get route_point => 'Punto de ruta';

  @override
  String get russian => 'Ruso';

  @override
  String get save => 'Guardar';

  @override
  String get save_list => 'Guardar Lista';

  @override
  String get save_trail => 'Guardar Ruta';

  @override
  String get save_your_trail_first => 'Guarda tu ruta primero';

  @override
  String get search => 'Search';

  @override
  String get search_cities => 'Buscar ciudades';

  @override
  String get search_for_trails_places => 'Busca rutas, lugares';

  @override
  String get search_list => 'Search list';

  @override
  String get search_places => 'Buscar lugares';

  @override
  String get search_trails => 'Buscar ruta';

  @override
  String get select_date => 'Select date';

  @override
  String get select_list => 'Seleccionar Lista';

  @override
  String get selected => 'seleccionado';

  @override
  String get send_to => 'Send to...';

  @override
  String get set_private => 'Set private';

  @override
  String get set_public => 'Set public';

  @override
  String get settings => 'Configuración';

  @override
  String get settings_notification_comment_mention =>
      'Alguien te mencionó en un comentario';

  @override
  String get settings_notification_list_share =>
      'Alguien ha compartido una lista contigo';

  @override
  String get settings_notification_new_follower => 'Tienes un nuevo seguidor';

  @override
  String get settings_notification_summit_log_create =>
      'Alguien ha creado un registro de cumbre en tu ruta';

  @override
  String get settings_notification_summit_log_mention =>
      'Alguien te ha mencionado en un registro de cumbre';

  @override
  String get settings_notification_trail_comment =>
      'Alguien ha dejado un comentario sobre tu ruta';

  @override
  String get settings_notification_trail_like => 'Somone liked your trail';

  @override
  String get settings_notification_trail_mention =>
      'Alguien te mencionó en una ruta';

  @override
  String get settings_notification_trail_share =>
      'Alguien ha compartido una ruta contigo';

  @override
  String get settings_privacy_account_private =>
      'Solo tu puedes ver tu perfil. No aparecerás en los resultados de búsquedas. Otros usuarios no pueden seguirte o compartir rutas contigo. Todavía sí puedes publicar tus rutas o listas.';

  @override
  String get settings_privacy_account_public =>
      'Todos pueden ver tu perfil. Aparecerás en los resultados de búsquedas. Otros usuarios pueden seguirte y compartir contigo rutas.';

  @override
  String get settings_privacy_lists_private =>
      'Tus listas son privadas por defecto. Nadie excepto tú podrás verlas. Puedes cambiar esta configuración en cualquier momento para listas específicas.';

  @override
  String get settings_privacy_lists_public =>
      'Tus listas son públicas por defecto. Todos podrán verlas. Puedes cambiar esta configuración en cualquier momento para listas específicas.';

  @override
  String get settings_privacy_trails_private =>
      'Tus rutas son privadas por defecto. Nadie excepto tú podrás verlas. Puedes cambiar esta configuración en cualquier momento para rutas específicas.';

  @override
  String get settings_privacy_trails_public =>
      'Tus rutas son públicas por defecto. Todos podrán verlas. Puedes cambiar esta configuración en cualquier momento para rutas específicas.';

  @override
  String get settings_saved => 'Configuración guardada';

  @override
  String get share => 'Compartir';

  @override
  String get share_profile => 'Compartir perfil';

  @override
  String get share_this_list => 'Compartir esta lista';

  @override
  String get share_this_trail => 'Comparte esta ruta';

  @override
  String get shared => 'Compartido';

  @override
  String get shared_by => 'Compartido por';

  @override
  String get shared_with => 'Compartido con';

  @override
  String get shelter => 'Refugio';

  @override
  String get shortest => 'más corto';

  @override
  String get show_in_overview => 'Mostrar en la panorámica';

  @override
  String get show_less => 'Mostrar menos';

  @override
  String get show_on_map => 'Mostrar en mapa';

  @override
  String get shower => 'Ducha';

  @override
  String get skiing => 'Esquí';

  @override
  String get slogan => '¡Guarda tus aventuras!';

  @override
  String get slope => 'Pendiente';

  @override
  String get someone => 'Alguien';

  @override
  String get sort => 'Ordenar';

  @override
  String get spanish => 'Español';

  @override
  String get speed => 'Velocidad';

  @override
  String get start => 'Empezar';

  @override
  String get statistics => 'Estadísticas';

  @override
  String get stop_drawing => 'Parar de diseñar';

  @override
  String get stop_editing => 'Parar de editar';

  @override
  String get strava_integration_after_date_hint =>
      'Si tu cuenta tiene una gran cantidad de actividades, es posible que alcances el límite de peticiones de la API de Strava, lo que impedirá la sincronización de todas las actividades a la vez. Para mitigar este problema, puedes establecer una fecha \"Posterior a\" a continuación, de modo que solo se sincronicen las actividades que se registraron después de esa fecha.';

  @override
  String get subcategories => 'Subcategorías';

  @override
  String get subway_stop => 'Entrada de metro';

  @override
  String get summit => 'Cumbre';

  @override
  String get summit_book => 'Libro de ascensos';

  @override
  String get table => 'Tabla';

  @override
  String get tags => 'Etiquetas';

  @override
  String get text => 'Texto';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get tilesets => 'Ficha personalizada';

  @override
  String get time => 'Time';

  @override
  String get toilets => 'Baños';

  @override
  String get top_speed => 'Velocidad máxima';

  @override
  String get tourism => 'Turismo';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Rutas',
      one: 'Ruta',
    );
    return '$_temp0';
  }

  @override
  String get trail_copied_successfully => 'trail copied successfully';

  @override
  String get trail_has_no_gpx => 'This trail has no GPX data.';

  @override
  String get trail_not_in_list => 'Trail is not in any list';

  @override
  String get trail_not_shared => 'No compartida con nadie';

  @override
  String get trail_saved_successfully => 'Ruta guardada con éxito';

  @override
  String get some_waypoints_failed_to_save =>
      'Trail saved, but some waypoints failed to save';

  @override
  String get trails_for_you => 'Rutas para ti';

  @override
  String get tram_stop => 'Parada de tranvía';

  @override
  String get unchanged => 'sin cambios';

  @override
  String get units => 'Unidades';

  @override
  String get unlink => 'Desconectar';

  @override
  String get upload_file => 'Cargar archivo';

  @override
  String get upload_gpx => 'Cargar GPX';

  @override
  String get upload_new_file => 'Subir nuevo archivo';

  @override
  String get uploaded => 'cargado';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Successfully uploaded trail to Hammerhead';

  @override
  String get use_hills => 'Usar cuestas';

  @override
  String get use_roads => 'Utilizar carreteras';

  @override
  String get users => 'Users';

  @override
  String get username => 'Nombre de usuario';

  @override
  String get username_not_unique =>
      'This username is already taken. Please try another.';

  @override
  String get view => 'Ver';

  @override
  String get viewpoint => 'Mirador';

  @override
  String get visibilty => 'Visibilidad';

  @override
  String get visibilty_status => 'Estado de visibilidad';

  @override
  String get walking_speed => 'Velocidad al caminar';

  @override
  String get water => 'Agua';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Puntos de Interés',
      one: 'Punto de Interés',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get width => 'Anchura';

  @override
  String get wrong_username_or_password => 'Usuario o contraseña no correctas';

  @override
  String get you_can => 'Puedes';

  @override
  String get you_have_arrived => 'You\'ve arrived';

  @override
  String get settings_categories_confirm_disable_title => 'Hide this category?';

  @override
  String get settings_categories_confirm_disable_subcategory_title =>
      'Hide this subcategory?';

  @override
  String settings_categories_confirm_disable_body(int count) {
    return '$count of your trails use this category. They will stay published but this filter will be hidden.';
  }

  @override
  String get settings_categories_confirm_view_trails => 'View trails';

  @override
  String get settings_categories_confirm_disable_confirm => 'Disable anyway';

  @override
  String get settings_categories_empty_title => 'No subcategories';

  @override
  String get settings_categories_empty_body =>
      'This category has no subcategories to configure.';

  @override
  String get settings_categories_reorder_hint =>
      'Categories control which trail types you see and in what order. Turn one off to hide it as a filter — your trails stay published, they just won\'t appear under that category. Tap a category to manage its subcategories individually.\n\nTo change the order, press and hold a row, then drag it to a new position. The order you set here is reflected everywhere categories are shown.';
}
