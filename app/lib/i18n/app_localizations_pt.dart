// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Portuguese (`pt`).
class AppLocalizationsPt extends AppLocalizations {
  AppLocalizationsPt([String locale = 'pt']) : super(locale);

  @override
  String get about => 'Sobre';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Está prestes a excluir a sua conta. Todos os seus percursos também serão excluídos. Quer continuar?';

  @override
  String get account_privacy => 'Account privacy';

  @override
  String get add_bio => 'Add Bio';

  @override
  String get add_waypoint => 'Adicionar ponto de vista';

  @override
  String get adjust_track => 'Adjust track';

  @override
  String get after => 'Depois';

  @override
  String get all => 'All';

  @override
  String get altitude => 'Altitude';

  @override
  String get author => 'Author';

  @override
  String get average_speed => 'Vel. média';

  @override
  String get background_location_body =>
      'wanderer collects location data in the background so your trail keeps recording when the screen is off or the app is closed. Your recorded track stays on your device until you choose to save the trail.\n\nAndroid only offers this in system settings: open Location and choose \"Allow all the time\".';

  @override
  String get background_location_confirm => 'Open settings';

  @override
  String get background_location_title => 'Keep recording in the background';

  @override
  String get basic_info => 'Informações básicas';

  @override
  String get before => 'Antes';

  @override
  String get by => 'by';

  @override
  String get cancel => 'Cancelar';

  @override
  String get discard => 'Discard';

  @override
  String get discard_trail_confirm => 'Discard this trail and its changes?';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Cartões',
      one: 'Cartão',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Categorias';

  @override
  String get category => 'Categoria';

  @override
  String get change_email => 'Mudar correio eletrónico';

  @override
  String get change_password => 'Mudar senha';

  @override
  String get clear_all => 'Clear all';

  @override
  String get close => 'Fechar';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Comentários',
      one: 'Comentário',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Completada';

  @override
  String get completion_status => 'Estado de conclusão';

  @override
  String get confirm_deletion => 'Confirmar eliminação';

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
  String get copy_link => 'Copiar ligação';

  @override
  String get create_waypoint => 'Create waypoint';

  @override
  String get creation_date => 'Data de criação';

  @override
  String get current_password => 'Senha atual';

  @override
  String get danger_zone => 'Zona de perigo';

  @override
  String get date => 'Data';

  @override
  String get delete => 'Excluir';

  @override
  String get not_now => 'Not now';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Excluir conta';

  @override
  String get delete_trail_confirm =>
      'Você realmente quer excluir esta trilha? Esta ação não pode ser desfeita.';

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
  String get description => 'Descrição';

  @override
  String get difficult => 'Difícil';

  @override
  String get difficulty => 'Dificuldade';

  @override
  String get directions => 'Instruções';

  @override
  String get distance => 'Distância';

  @override
  String get download => 'Download';

  @override
  String get duration => 'Duração';

  @override
  String get easy => 'Fácil';

  @override
  String get edit => 'Editar';

  @override
  String get edit_needs_connection =>
      'Editing works on the server copy of this trail. Connect to the internet to edit it.';

  @override
  String get edit_waypoint => 'Editar ponto de passagem';

  @override
  String get edited => 'edited';

  @override
  String get elevation_gain => 'Ganho de elevação';

  @override
  String get elevation_loss => 'Perda de elevação';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => 'Correio eletrónico';

  @override
  String get email_not_unique => 'This email address is already in use.';

  @override
  String get email_updated => 'Email updated';

  @override
  String get error_deleting_trail => 'Error deleting trail';

  @override
  String get error_reading_file => 'Erro ao ler o arquivo';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Erro ao gravar percurso';

  @override
  String get error_updating_password => 'Erro ao atualizar password';

  @override
  String get explore => 'Explorar';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get stop_recording => 'Parar gravação';

  @override
  String get stop_recording_confirm => 'Parar a gravação?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get filter_tags => 'Filter tags';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => 'Finish';

  @override
  String get finish_disabled_hint =>
      'Add at least 2 anchors to finish your route.';

  @override
  String get follow => 'Follow';

  @override
  String get followers => 'Followers';

  @override
  String get following => 'Following';

  @override
  String get from_photos => 'From Photos';

  @override
  String get help => 'Ajuda';

  @override
  String get hiking => 'Montanhismo';

  @override
  String get home => 'Home';

  @override
  String get icon => 'Ícone';

  @override
  String get imperial => 'Imperial';

  @override
  String get joined => 'Joined';

  @override
  String get language => 'Língua';

  @override
  String get latitude => 'Latitude';

  @override
  String get like_status => 'Like Status';

  @override
  String get liked => 'Liked';

  @override
  String get likes => 'Likes';

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
  String get location => 'Localização';

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
  String get login => 'Login';

  @override
  String get logout => 'Sair';

  @override
  String get longitude => 'Longitude';

  @override
  String get map => 'Mapa';

  @override
  String get metric => 'Métrica';

  @override
  String get moderate => 'Moderado';

  @override
  String get my_account => 'A minha conta';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String get name => 'Nome';

  @override
  String get navigate => 'Navigate';

  @override
  String get new_password => 'Nova senha';

  @override
  String get new_password_success => 'The new password has been set';

  @override
  String get new_trail => 'Nova trilha';

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
  String get no_comments_so_far => 'No comments so far';

  @override
  String get no_data => 'Sem dados';

  @override
  String get no_description_for_now => 'No description for now';

  @override
  String get no_gps_data_in_image => 'No GPS data in image';

  @override
  String get no_preference => 'Nenhuma preferência';

  @override
  String get no_trails_found => 'No trails found';

  @override
  String get not_completed => 'Não preenchido';

  @override
  String get notifications => 'Notifications';

  @override
  String get only_me => 'Only me';

  @override
  String get or => 'ou';

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
  String get password => 'Senha';

  @override
  String get password_confirm => 'Confirm password';

  @override
  String get passwords_must_match => 'Passwords must match';

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
  String get privacy => 'Privacy';

  @override
  String get private => 'Private';

  @override
  String get profile => 'Perfil';

  @override
  String get public => 'Público';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get register => 'Registo';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Reset';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => 'Retomar a gravação?';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Routes',
      one: 'Route',
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
  String get search_for_trails_places => 'Procurar trilhos, locais';

  @override
  String get select_date => 'Select date';

  @override
  String get settings => 'Definições';

  @override
  String get settings_notification_comment_mention =>
      'Someone mentioned you in a comment';

  @override
  String get settings_notification_list_share =>
      'Someone shared a list with you';

  @override
  String get settings_notification_new_follower => 'You have a new follower';

  @override
  String get settings_notification_summit_log_create =>
      'Someone created a summit log on your trail';

  @override
  String get settings_notification_summit_log_mention =>
      'Someone mentioned you in a summit log';

  @override
  String get settings_notification_trail_comment =>
      'Someone left a comment on your trail';

  @override
  String get settings_notification_trail_like => 'Somone liked your trail';

  @override
  String get settings_notification_trail_mention =>
      'Someone mentioned you in a trail';

  @override
  String get settings_notification_trail_share =>
      'Someone shared a trail with you';

  @override
  String get settings_privacy_account_private =>
      'Only you can see your profile. You will not appear in search results. Other users cannot follow you or share trails with you. You can still publish trails or lists.';

  @override
  String get settings_privacy_account_public =>
      'Everyone can see your profile. You appear in search results. Other users can follow you and share trails with you.';

  @override
  String get settings_privacy_lists_private =>
      'Your lists are private by default. No one except you will be able to see them. You can change this setting at any point for individual lists.';

  @override
  String get settings_privacy_lists_public =>
      'Your lists are public by default. Everyone will be able to see them. You can change this setting at any point for individual lists.';

  @override
  String get settings_privacy_trails_private =>
      'Your trails are private by default. No one except you will be able to see them. You can change this setting at any point for individual trails.';

  @override
  String get settings_privacy_trails_public =>
      'Your trails are public by default. Everyone will be able to see them. You can change this setting at any point for individual trails.';

  @override
  String get share => 'Partilhar';

  @override
  String get share_profile => 'Share profile';

  @override
  String get shared => 'Shared';

  @override
  String get show_on_map => 'Mostrar no mapa';

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
  String get slogan => 'Os seus percursos. Os seus dados. O seu servidor.';

  @override
  String get sort => 'Ordenar';

  @override
  String get speed => 'Velocidade';

  @override
  String get start => 'Start';

  @override
  String get subcategories => 'Subcategorias';

  @override
  String get summit_book => 'Livro da cimeira';

  @override
  String get sync_failed => 'Upload failed · Tap to retry';

  @override
  String get sync_pending => 'Waiting to upload';

  @override
  String get sync_uploading => 'Uploading…';

  @override
  String get tags => 'Tags';

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
      other: 'Percursos',
      one: 'Percurso',
    );
    return '$_temp0';
  }

  @override
  String get trail_saved_successfully => 'Percurso gravado com sucesso';

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
  String get username => 'Nome de utilizador';

  @override
  String get username_not_unique =>
      'This username is already taken. Please try another.';

  @override
  String get visibilty_status => 'Visibility status';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Pontos de passagem',
      one: 'Ponto de passagem',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get wrong_username_or_password =>
      'Nome de utilizador ou palavra-passe errados';

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
  String get open_in_new_tab => 'Open in new tab';

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
  String get reverse_direction => 'Reverse direction';

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
  String get edit_route => 'Edit route';

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
