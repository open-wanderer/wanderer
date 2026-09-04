// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Spanish Castilian (`es`).
class AppLocalizationsEs extends AppLocalizations {
  AppLocalizationsEs([String locale = 'es']) : super(locale);

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
  String get add_bio => 'Añadir biografía';

  @override
  String get add_waypoint => 'Añadir Punto de Interés';

  @override
  String get adjust_track => 'Adjust track';

  @override
  String get after => 'Después';

  @override
  String get all => 'All';

  @override
  String get altitude => 'Altitud';

  @override
  String get author => 'Autor';

  @override
  String get average_speed => 'Velocidad media';

  @override
  String get background_location_body =>
      'wanderer collects location data in the background so your trail keeps recording when the screen is off or the app is closed. Without this, clearing wanderer from your recent apps ends the recording. Location is only collected while a trail is in progress, and is saved with that trail.';

  @override
  String get background_location_confirm => 'Continue';

  @override
  String get background_location_title => 'Keep recording in the background';

  @override
  String get basic_info => 'Información básica';

  @override
  String get before => 'Antes';

  @override
  String get by => 'de';

  @override
  String get cancel => 'Borrar';

  @override
  String get discard => 'Discard';

  @override
  String get discard_trail_confirm => 'Discard this trail and its changes?';

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
  String get change_email => 'Cambiar mail';

  @override
  String get change_password => 'Cambiar password';

  @override
  String get clear_all => 'Limpiar todo';

  @override
  String get close => 'Cerrar';

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
  String get completion_status => 'Estado de avance';

  @override
  String get confirm_deletion => 'Confirmar Borrado';

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
  String get location_unavailable =>
      'Unable to determine your location. Please try again.';

  @override
  String get copy_link => 'Copiar Enlace';

  @override
  String get create_waypoint => 'Crear punto de ruta';

  @override
  String get creation_date => 'Fecha de creación';

  @override
  String get current_password => 'Contraseña actual';

  @override
  String get danger_zone => 'Zona peligrosa';

  @override
  String get date => 'Fecha';

  @override
  String get delete => 'Borrar';

  @override
  String get not_now => 'Not now';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Borrar Cuenta';

  @override
  String get delete_trail_confirm =>
      '¿Seguro que quieres borrar esta ruta? Esta acción no puede restablecerse.';

  @override
  String get delete_blocked_while_uploading =>
      'This trail is uploading right now. Wait for the upload to finish, then try again.';

  @override
  String get delete_unsynced_trail_confirm =>
      'Delete this trail? It hasn\'t been uploaded yet, so this can\'t be undone.';

  @override
  String get delete_needs_connection =>
      'This trail is already on the server. Connect to the internet to delete it.';

  @override
  String get description => 'Descripción';

  @override
  String get difficult => 'Dificultad';

  @override
  String get difficulty => 'Dificultad';

  @override
  String get directions => 'Indicaciones';

  @override
  String get distance => 'Distancia';

  @override
  String get download => 'Descarga';

  @override
  String get duration => 'Duración';

  @override
  String get easy => 'Fácil';

  @override
  String get edit => 'Editar';

  @override
  String get edit_needs_connection =>
      'Editing works on the server copy of this trail. Connect to the internet to edit it.';

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
  String get email_not_unique =>
      'Esta dirección de correo electrónico ya se encuentra en uso.';

  @override
  String get email_updated => 'Correo electrónico actualizado';

  @override
  String get error_deleting_trail => 'Error deleting trail';

  @override
  String get error_reading_file => 'Error leyendo el archivo';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Error guardando la ruta';

  @override
  String get error_updating_password => 'Error actualizando la contraseña';

  @override
  String get explore => 'Explora';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get stop_recording => 'Detener grabación';

  @override
  String get stop_recording_confirm => '¿Detener la grabación?';

  @override
  String get search_this_area => 'Search this area';

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
  String get follow => 'Seguir';

  @override
  String get followers => 'Seguidores';

  @override
  String get following => 'Siguiendo';

  @override
  String get from_photos => 'Desde fotos';

  @override
  String get help => 'Ayuda';

  @override
  String get hiking => 'Senderismo';

  @override
  String get home => 'Inicio';

  @override
  String get icon => 'Icono';

  @override
  String get imperial => 'Imperial';

  @override
  String get joined => 'Afiliado';

  @override
  String get language => 'Idioma';

  @override
  String get latitude => 'Latitud';

  @override
  String get like_status => 'Estado \"Me gusta\"';

  @override
  String get liked => 'Me gusta';

  @override
  String get likes => 'Me gusta';

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
  String get location => 'Ubicación';

  @override
  String get locations => 'Locations';

  @override
  String get center_on_my_location => 'Center on my location';

  @override
  String get location_tracking_notification_title => 'wanderer';

  @override
  String get location_tracking_notification_text => 'Recording your trail';

  @override
  String location_tracking_notification_text_navigating(String trail) {
    return 'Navigating $trail';
  }

  @override
  String get login => 'Acceso';

  @override
  String get logout => 'Salir';

  @override
  String get longitude => 'Longitud';

  @override
  String get map => 'Mapa';

  @override
  String get metric => 'Métrica';

  @override
  String get moderate => 'Medio';

  @override
  String get my_account => 'Mi cuenta';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String get name => 'Nombre';

  @override
  String get navigate => 'Navigate';

  @override
  String get new_password => 'Nueva contraseña';

  @override
  String get new_password_success => 'La nueva contraseña ha sido configurada';

  @override
  String get new_trail => 'Nueva Ruta';

  @override
  String get trail_source_planner => 'Open trail planner';

  @override
  String get trail_source_planner_description =>
      'Draw a new route on the map, waypoint by waypoint.';

  @override
  String get trail_source_record => 'Record trail';

  @override
  String get trail_source_record_description =>
      'Track your live coordinates and log your journey in real-time.';

  @override
  String get trail_source_import => 'Import file';

  @override
  String get trail_source_import_description =>
      'Upload GPX, KML, KMZ, TCX or FIT files directly from your device storage.';

  @override
  String get trail_source_import_error => 'Could not import file';

  @override
  String get trail_source_offline_import_error =>
      'Only GPX files can be imported offline';

  @override
  String get no_comments_so_far => 'Ningún comentario todavía';

  @override
  String get no_data => 'No datos';

  @override
  String get no_description_for_now => 'Ninguna descripción de momento';

  @override
  String get no_gps_data_in_image => 'Sin datos GPS en la imagen';

  @override
  String get no_preference => 'Ninguna preferencia';

  @override
  String get no_trails_found => 'No trails found';

  @override
  String get not_completed => 'No completado';

  @override
  String get notifications => 'Notificaciones';

  @override
  String get only_me => 'Solo yo';

  @override
  String get or => 'o';

  @override
  String get own_trails_empty_body =>
      'Trails you record or save offline appear here, and upload automatically once you\'re back online.';

  @override
  String get own_trails_empty_title => 'Nothing saved yet';

  @override
  String get own_trails_offline_banner =>
      'Offline — showing only trails on this device.';

  @override
  String get trails_on_device => 'Trails (on device)';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Contraseña';

  @override
  String get password_confirm => 'Confirmar contraseña';

  @override
  String get passwords_must_match => 'Las contraseñas tienen que coincidir';

  @override
  String photo_copy_failed_toast(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Trail saved, but $count photos couldn\'t be saved.',
      one: 'Trail saved, but 1 photo couldn\'t be saved.',
    );
    return '$_temp0';
  }

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
  String get privacy => 'Privacidad';

  @override
  String get private => 'Privado';

  @override
  String get profile => 'Perfil';

  @override
  String get public => 'Público';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get register => 'Registrar';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Restablecer';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => '¿Reanudar la grabación?';

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
  String get follow_roads => 'Follow roads';

  @override
  String get follow_roads_description =>
      'Snap the recorded path to the nearest roads and trails.';

  @override
  String get recalculate_heights => 'Recalculate heights';

  @override
  String get recalculate_heights_description =>
      'Replace recorded GPS elevation with more accurate values from the map.';

  @override
  String get save => 'Guardar';

  @override
  String get save_recording_options => 'Save recording';

  @override
  String get save_track => 'Save track';

  @override
  String get search => 'Search';

  @override
  String get search_for_trails_places => 'Busca rutas, lugares';

  @override
  String get select_date => 'Select date';

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
  String get share => 'Compartir';

  @override
  String get share_profile => 'Compartir perfil';

  @override
  String get shared => 'Compartido';

  @override
  String get show_on_map => 'Mostrar en mapa';

  @override
  String signout_unsynced_warning(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'You have $count trails not uploaded yet. Signing out won\'t delete them — they\'ll be here when you sign back in — but they won\'t upload until then.',
      one:
          'You have 1 trail not uploaded yet. Signing out won\'t delete it — it\'ll be here when you sign back in — but it won\'t upload until then.',
    );
    return '$_temp0';
  }

  @override
  String get slogan => 'Tus rutas. Tus datos. Tu servidor.';

  @override
  String get sort => 'Ordenar';

  @override
  String get speed => 'Velocidad';

  @override
  String get start => 'Empezar';

  @override
  String get subcategories => 'Subcategorías';

  @override
  String get summit_book => 'Libro de ascensos';

  @override
  String get sync_failed => 'Upload failed · Tap to retry';

  @override
  String get sync_pending => 'Waiting to upload';

  @override
  String get sync_uploading => 'Uploading…';

  @override
  String get tags => 'Etiquetas';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get time_in_motion => 'Time in Motion';

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
  String get trail_saved_successfully => 'Ruta guardada con éxito';

  @override
  String get trail_not_on_this_device =>
      'This trail is no longer on this device.';

  @override
  String get trail_uploaded_reopen_to_edit =>
      'This trail finished uploading. Re-open it from your trails to keep editing.';

  @override
  String get some_waypoints_failed_to_save =>
      'Trail saved, but some waypoints failed to save';

  @override
  String get units => 'Unidades';

  @override
  String get users => 'Users';

  @override
  String get username => 'Nombre de usuario';

  @override
  String get username_not_unique =>
      'Este nombre de usuario ya está en uso. Prueba con otro.';

  @override
  String get visibilty_status => 'Estado de visibilidad';

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
  String get wrong_username_or_password => 'Usuario o contraseña no correctas';

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

  @override
  String get something_went_wrong => 'Something went wrong';

  @override
  String get technical_details => 'Technical Details';

  @override
  String get link => 'Link';

  @override
  String get url => 'URL';

  @override
  String get open_in_new_tab => 'Abrir en nueva pestaña';

  @override
  String get remove => 'Remove';

  @override
  String get remove_download_confirm_body =>
      'This removes the downloaded copy from this device. The trail itself is not deleted — you\'ll need to download it again to use it offline.';

  @override
  String get apply => 'Apply';

  @override
  String get add_at_least_2_anchors_hint =>
      'Add at least 2 anchors to see the elevation profile.';

  @override
  String get reverse_direction => 'Invertir la dirección';

  @override
  String get delete_all => 'Delete all';

  @override
  String get auto_routing => 'Auto-routing';

  @override
  String get auto_routing_hint =>
      'Automatically follow roads and paths between anchors.';

  @override
  String get travel_profile => 'Travel profile';

  @override
  String get no_track_data => 'No track data';

  @override
  String get available_offline => 'Available offline';

  @override
  String get no_lists_found => 'No lists found';

  @override
  String get search_lists => 'Search lists…';

  @override
  String get search_for_a_location => 'Search for a location';

  @override
  String no_results_for_query(String query) {
    return 'No results for \"$query\"';
  }

  @override
  String get filter => 'Filter';

  @override
  String no_label_yet(String label) {
    return 'No $label yet.';
  }

  @override
  String get no_lists_yet => 'No lists yet.';

  @override
  String get no_bio_yet => 'No bio yet.';

  @override
  String get show_more => 'Show more';

  @override
  String get show_less => 'Show less';

  @override
  String get feed => 'Feed';

  @override
  String get no_trails_yet => 'No trails yet.';

  @override
  String get library_empty_title => 'No downloaded trails';

  @override
  String get library_empty_body =>
      'Trails you download are kept here so you can open them offline.';

  @override
  String get library_empty_search_body =>
      'Try a different search term or clear your filters.';

  @override
  String get search_location => 'Search location';

  @override
  String no_servers_match_query(String query) {
    return 'No servers match \"$query\"';
  }

  @override
  String get use_custom_url_instead => 'Use custom URL instead';

  @override
  String get select_instance => 'Select Instance';

  @override
  String get enter_server_url_hint => 'Enter server URL (e.g. wanderer.to)';

  @override
  String get search_library => 'Search library…';

  @override
  String language_and_units(String language, String units) {
    return '$language & $units';
  }

  @override
  String get edit_route => 'Editar ruta';

  @override
  String get undo => 'Undo';

  @override
  String get redo => 'Redo';

  @override
  String get bold => 'Bold';

  @override
  String get italic => 'Italic';

  @override
  String get underline => 'Underline';

  @override
  String get bullet_list => 'Bullet list';

  @override
  String get ordered_list => 'Ordered list';

  @override
  String get blockquote => 'Blockquote';

  @override
  String get library => 'Library';

  @override
  String get settings_offline_regions_title => 'Offline Maps/Regions';

  @override
  String get regions_search_hint => 'Search regions';

  @override
  String get regions_dem_toggle_label => 'Download elevation data (DEM)';

  @override
  String get regions_dem_toggle_caption =>
      'Adds hillshading; increases download size';

  @override
  String get regions_update_available => 'Update available';

  @override
  String get regions_update_action => 'Update';

  @override
  String get regions_retry => 'Retry';

  @override
  String get regions_not_yet_available => 'Not yet available';

  @override
  String get regions_build_failed => 'Build failed';

  @override
  String regions_delete_confirm_title(String name) {
    return 'Delete $name?';
  }

  @override
  String get regions_delete_confirm_body =>
      'This removes the downloaded map and elevation data for this region. You\'ll need to download it again to use it offline.';

  @override
  String get regions_delete_confirm_action => 'Delete';

  @override
  String regions_disk_usage_summary(String size, num count) {
    return '$size used across $count downloaded region(s)';
  }

  @override
  String get regions_empty_search_title => 'No matching regions';

  @override
  String get regions_empty_search_body => 'Try a different search term.';

  @override
  String get regions_empty_catalog_title => 'No offline regions available';

  @override
  String get regions_empty_catalog_body =>
      'Ask your Wanderer instance administrator to configure downloadable regions.';

  @override
  String get regions_vector_tile_title => 'Vector';

  @override
  String get regions_dem_tile_title => 'Elevation data';

  @override
  String get regions_download_failed => 'Download failed';

  @override
  String get regions_dem_locked_subtitle => 'Download map data first';

  @override
  String get regions_offline_unavailable_title => 'Can\'t load regions';

  @override
  String get regions_offline_unavailable_body =>
      'Connect to the internet to browse and manage downloadable regions.';

  @override
  String get regions_map_geometry_failed => 'Could not load region outline';

  @override
  String get regions_map_back_label => 'Back to regions';

  @override
  String regions_group_expand_label(String name) {
    return 'Expand $name';
  }

  @override
  String regions_group_collapse_label(String name) {
    return 'Collapse $name';
  }

  @override
  String get offline_title => 'You\'re offline';

  @override
  String get offline_try_again => 'Try again';

  @override
  String get offline_map_body =>
      'Connect to the internet to load the map. Downloaded trails are still available.';

  @override
  String get offline_list_body => 'Connect to the internet to load lists.';

  @override
  String get offline_profile_body =>
      'Connect to the internet to load your full profile.';

  @override
  String get offline_settings_banner =>
      'You\'re offline. Settings are read-only until you reconnect.';

  @override
  String get offline_action_unavailable =>
      'You\'re offline — try again once you\'re back online.';

  @override
  String get offline_categories_body =>
      'Connect to the internet to manage categories.';

  @override
  String get offline_trail_search_body =>
      'Connect to the internet to search for trails. Downloaded trails are still available.';
}
