// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Portuguese (`pt`).
class AppLocalizationsPt extends AppLocalizations {
  AppLocalizationsPt([String locale = 'pt']) : super(locale);

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
      'Está prestes a excluir a sua conta. Todos os seus percursos também serão excluídos. Quer continuar?';

  @override
  String get account_privacy => 'Account privacy';

  @override
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Atividades',
      one: 'Atividade',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => 'Add Bio';

  @override
  String get add_entry => 'Adicionar entrada';

  @override
  String get add_to_list => 'Adicionar à lista';

  @override
  String get add_waypoint => 'Adicionar ponto de vista';

  @override
  String get added_trail_to => 'Trilha adicionada para';

  @override
  String get added_trails_to => 'trilhas adicionada para';

  @override
  String get after => 'Depois';

  @override
  String get all => 'All';

  @override
  String get all_activities => 'Todas as atividades';

  @override
  String get allow_auto_geolocate =>
      'Begin drawing a new trail from your current location';

  @override
  String get alphabetical => 'Alfabético';

  @override
  String get already_account => 'Já tem uma conta?';

  @override
  String get altitude => 'Altitude';

  @override
  String get amenity => 'Amenity';

  @override
  String get api_documentation => 'Documentação da API';

  @override
  String get api_tokens => 'API Tokens';

  @override
  String get api_tokens_hint =>
      'API Tokens can be used to grant 3rd party applications access to your wanderer account.';

  @override
  String get apply_user_settings => 'Apply user settings';

  @override
  String get attraction => 'Attraction';

  @override
  String get author => 'Author';

  @override
  String get avatar => 'Avatar';

  @override
  String get average_speed => 'Vel. média';

  @override
  String get avoid_bad_surfaces => 'Avoid Bad Surfaces';

  @override
  String get back => 'Back';

  @override
  String get back_to_login => 'Back to Login';

  @override
  String get bakery => 'Bakery';

  @override
  String get barrier => 'Barrier';

  @override
  String get basic_info => 'Informações básicas';

  @override
  String get basque => 'Basque';

  @override
  String get before => 'Antes';

  @override
  String get behavior => 'Behavior';

  @override
  String get bicycle_parking => 'Bicycle Parking';

  @override
  String get bicycle_rental => 'Bicycle Rental';

  @override
  String get bicycle_shop => 'Bicycle Shop';

  @override
  String get bike_type => 'Bike Type';

  @override
  String get bus_stop => 'Bus stop';

  @override
  String get by => 'by';

  @override
  String get campsite => 'Campsite';

  @override
  String get can => 'pode';

  @override
  String get cancel => 'Cancelar';

  @override
  String get car => 'Car';

  @override
  String get car_motorcycle => 'Car/Motorcycle';

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
  String get change => 'Alterar';

  @override
  String get change_email => 'Mudar correio eletrónico';

  @override
  String get change_password => 'Mudar senha';

  @override
  String get changelog => 'Registo de alterações';

  @override
  String get chinese => 'Chinês (simplificado)';

  @override
  String get clear_all => 'Clear all';

  @override
  String get climbing => 'Climbing';

  @override
  String get close => 'Fechar';

  @override
  String get collapse_trail_list => 'Collapse trail list';

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
  String get completed_a_trail => 'completed a trail';

  @override
  String get completed_tours => 'Completed tours';

  @override
  String get completion_status => 'Estado de conclusão';

  @override
  String get confirm => 'Confirmar';

  @override
  String get confirm_deletion => 'Confirmar eliminação';

  @override
  String get confirm_publish => 'Confirm publishing';

  @override
  String get confirm_share => 'Confirmar partilha';

  @override
  String get connect => 'Connect';

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
  String get copy_link => 'Copiar ligação';

  @override
  String get create_new_list => 'Criar nova lista';

  @override
  String get create_waypoint => 'Create waypoint';

  @override
  String get creation_date => 'Data de criação';

  @override
  String get crop => 'Crop';

  @override
  String get cross => 'Cross';

  @override
  String get current_password => 'Senha atual';

  @override
  String get cycling => 'Ciclismo';

  @override
  String get cycling_speed => 'Cycling Speed';

  @override
  String get czech => 'Czech';

  @override
  String get danger_zone => 'Zona de perigo';

  @override
  String get date => 'Data';

  @override
  String get default_category => 'Categoria inicial';

  @override
  String get default_location => 'Localização padrão';

  @override
  String get degrees => 'Degrees';

  @override
  String get delete => 'Excluir';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Excluir conta';

  @override
  String get delete_list_confirm =>
      'Você quer mesmo excluir esta lista? As trilhas na lista ainda estarão disponíveis.';

  @override
  String get delete_summit_log_confirm =>
      'Do you really want to delete this summit log? This action cannot be undone.';

  @override
  String get delete_trail_confirm =>
      'Você realmente quer excluir esta trilha? Esta ação não pode ser desfeita.';

  @override
  String get describe_your_trail => 'Descreva sua trilha';

  @override
  String get description => 'Descrição';

  @override
  String get difficult => 'Difícil';

  @override
  String get difficulty => 'Dificuldade';

  @override
  String get directions => 'Instruções';

  @override
  String get display => 'Ecrã';

  @override
  String get display_as => 'Exibir como';

  @override
  String get distance => 'Distância';

  @override
  String get documentation => 'Documentação';

  @override
  String get download => 'Download';

  @override
  String get draw_a_route => 'Desenhar uma rota';

  @override
  String get driving => 'Conduzir';

  @override
  String get duplicate => 'Duplicate';

  @override
  String get duration => 'Duração';

  @override
  String get dutch => 'Holandês';

  @override
  String get easy => 'Fácil';

  @override
  String get edit => 'Editar';

  @override
  String get edit_entry => 'Editar entrada';

  @override
  String get edit_list => 'Editar lista';

  @override
  String get edit_route => 'Edit route';

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
  String get email_verified => 'Email verified';

  @override
  String empty_activities(Object username) {
    return '$username has no activity yet';
  }

  @override
  String empty_bio(Object username) {
    return '$username has not added a bio yet';
  }

  @override
  String get empty_feed => 'Your feed is empty';

  @override
  String get empty_feed_explanation =>
      'Activities by you or people you follow will appear here';

  @override
  String empty_lists(Object username) {
    return '$username has no public lists';
  }

  @override
  String get enable_auto_routing => 'Enable auto-routing';

  @override
  String get english => 'Inglês';

  @override
  String get entry => 'Entrada';

  @override
  String get error_copying_trail => 'Error copying trail';

  @override
  String get error_creating_user => 'Erro ao criar utilizador';

  @override
  String get error_deleting_token => 'Error deleting token';

  @override
  String get error_disabling_strava_integration =>
      'Error disabling strava integration';

  @override
  String get error_during_login => 'Erro durante o ‘login’';

  @override
  String get error_during_password_reset =>
      'Unable to send password reset email';

  @override
  String get error_exporting_trail => 'Erro na exportação do percurso';

  @override
  String get error_generating_token => 'Error generating token';

  @override
  String get error_liking_trail => 'Error liking trail';

  @override
  String get error_logging_in_to_hammerhead => 'Error logging in to Hammerhead';

  @override
  String get error_logging_in_to_komoot => 'Error logging in to komoot';

  @override
  String get error_posting_comment => 'Error posting comment';

  @override
  String get error_printing_map => 'Erro na impressão do mapa';

  @override
  String get error_reading_file => 'Erro ao ler o arquivo';

  @override
  String get error_saving_list => 'Erro ao gravar lista';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Erro ao gravar percurso';

  @override
  String error_setting_up_integration(Object provider) {
    return 'Error setting up strava integration';
  }

  @override
  String get error_updating_hammerhead_integration =>
      'Error updating Hammerhead integration';

  @override
  String get error_updating_komoot_integration =>
      'Error updating komoot integration';

  @override
  String get error_updating_password => 'Erro ao atualizar password';

  @override
  String get error_updating_strava_integration =>
      'Error updating komoot integration';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Error uploading trail to Hammerhead';

  @override
  String get est_duration => 'Duração prevista';

  @override
  String get everyone_with_the_link => 'Everyone with the link';

  @override
  String get expand_trail_list => 'Expand trail list';

  @override
  String get expiration => 'Expiration';

  @override
  String get expires => 'Expires';

  @override
  String get explore => 'Explorar';

  @override
  String get explore_some_trails => 'Explore algumas trilhas';

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
  String get export => 'Exportar';

  @override
  String get export_all_trails => 'Exportar todos os percursos';

  @override
  String get favourite_sport => 'Favourite sport';

  @override
  String get features => 'Características';

  @override
  String get ferry => 'Ferry';

  @override
  String get file_format => 'Formato do ficheiro';

  @override
  String file_too_big(Object file, Object size) {
    return 'File $file is too big (max. $size)';
  }

  @override
  String get filter_categories => 'Filtrar categorias';

  @override
  String get filter_difficulty => 'Filter difficulty';

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
  String get fixed_speed => 'Fixed Speed';

  @override
  String get focus_map_on => 'Centrar mapa em';

  @override
  String get follow => 'Follow';

  @override
  String get follow_request_pending => 'Request pending';

  @override
  String get followers => 'Followers';

  @override
  String get following => 'Following';

  @override
  String get food => 'Food';

  @override
  String get food_drinks => 'Food & Drinks';

  @override
  String get forgot_your_password => 'Forgot your password?';

  @override
  String get french => 'Francês';

  @override
  String get from_file => 'From file';

  @override
  String get from_photos => 'From Photos';

  @override
  String get from_url => 'From URL';

  @override
  String get garage => 'Garage';

  @override
  String get gas_station => 'Gas station';

  @override
  String get generate_new_token => 'Generate new token';

  @override
  String get german => 'Alemão';

  @override
  String get get_position_from_exif => 'Obter coordenadas dos dados EXIF';

  @override
  String get get_started => 'Get started';

  @override
  String get grid => 'Grelha';

  @override
  String get grocery_store => 'Grocery store';

  @override
  String get hammerhead_integration_after_date_hint =>
      'If your hammerhead account is already synced with other trail databases, such as komoot or Strava, start syncing your Hammerhead data may result in duplicates. To avoid this, you can set an start date below, meaning only activities recorded after this date will be synced.';

  @override
  String get heading => 'Heading';

  @override
  String get height => 'Height';

  @override
  String get help => 'Ajuda';

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
  String get hiking => 'Montanhismo';

  @override
  String get home => 'Home';

  @override
  String get hotel => 'Hotel';

  @override
  String get hungarian => 'Húngaro';

  @override
  String get hut => 'Hut';

  @override
  String get hybrid => 'Hybrid';

  @override
  String get icon => 'Ícone';

  @override
  String get ignore_trails_before_date => 'Ignore trails before this date';

  @override
  String get imperial => 'Imperial';

  @override
  String get import => 'Importar';

  @override
  String get import_hint =>
      'Selecionar ou arrastar ficheiros GPX, FIT, KML ou TCX para aqui...';

  @override
  String get include_description => 'Incluir descrição';

  @override
  String get include_waypoints => 'Incluir pontos de passagem';

  @override
  String get integration_description_hammerhead =>
      'Syncs your Hammerhead tours with wanderer in regular intervals.';

  @override
  String get integration_description_komoot =>
      'Syncs your komoot tours with wanderer in regular intervals.';

  @override
  String get integration_description_strava =>
      'Syncs your strava routes & activities with wanderer in regular intervals.';

  @override
  String get integration_disabled => 'integration disabled';

  @override
  String get integration_enabled => 'integration enabled';

  @override
  String get integration_privacy_hint_original =>
      'Imported trails will maintain the same visibility they have on the external platform. For example, if the original trail was public, it will be public in wanderer, even if trails are private by default according to your privacy settings.';

  @override
  String get integration_privacy_hint_user =>
      'The original trail\'s visibility is discarded. Instead, the local privacy settings for trails are applied to all imported trails.';

  @override
  String get integrations => 'Integrations';

  @override
  String get invalid_date => 'Data inválida';

  @override
  String get invalid_username => 'Nome de usuário inválido';

  @override
  String get italian => 'Italiano';

  @override
  String get joined => 'Joined';

  @override
  String get keep_original => 'Keep original';

  @override
  String get keep_private => 'Keep private';

  @override
  String get language => 'Língua';

  @override
  String get last_used => 'Last used';

  @override
  String get latitude => 'Latitude';

  @override
  String layer(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Layers',
      one: 'Layer',
    );
    return '$_temp0';
  }

  @override
  String get license => 'Licença';

  @override
  String get like_status => 'Like Status';

  @override
  String get liked => 'Liked';

  @override
  String get likes => 'Likes';

  @override
  String get limited => 'Limited';

  @override
  String get link_copied => 'Link copiado!';

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
  String get list_not_shared => 'Não partilhado com ninguém';

  @override
  String get list_public_warning =>
      'All trails in this list will become public.';

  @override
  String get list_saved_successfully => 'Lista gravada com sucesso';

  @override
  String get list_share_warning =>
      'Partilhar uma lista automaticamente partilha todos os percursos nela contidos.';

  @override
  String get list_share_warning_update =>
      'Percursos adicionados serão partilhados com todos os que têm acesso a esta lista.';

  @override
  String get location => 'Localização';

  @override
  String get locations => 'Locations';

  @override
  String get location_tracking_notification_title => 'Wanderer';

  @override
  String get location_tracking_notification_text => 'Recording your trail';

  @override
  String get login => 'Login';

  @override
  String get login_details => 'Detalhes de login';

  @override
  String get logout => 'Sair';

  @override
  String get longitude => 'Longitude';

  @override
  String get loop => 'Loop';

  @override
  String get make_one => 'Faz um!';

  @override
  String get make_thumbnail => 'Faça miniatura';

  @override
  String get map => 'Mapa';

  @override
  String get map_style => 'Map style';

  @override
  String get max_hiking_difficulty => 'Max. Hiking Difficulty';

  @override
  String get metric => 'Métrica';

  @override
  String get moderate => 'Moderado';

  @override
  String get more => 'More';

  @override
  String get more_route_settings => 'More route settings';

  @override
  String get mountain => 'Mountain';

  @override
  String get mountain_pass => 'Mountain pass';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return 'Deve ter pelo menos $n caracteres';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return 'Must be at most $n characters long';
  }

  @override
  String get my_account => 'A minha conta';

  @override
  String get my_trails => 'My trails';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return '$n dias atrás';
  }

  @override
  String n_hours_ago(Object n) {
    return '$n horas atrás';
  }

  @override
  String n_minutes_ago(Object n) {
    return '$n minutos atrás';
  }

  @override
  String n_months_ago(Object n) {
    return '$n meses atrás';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n seconds ago',
      zero: 'just now',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return '$n anos atrás';
  }

  @override
  String get name => 'Nome';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => 'Próximo';

  @override
  String get never => 'Never';

  @override
  String get new_list => 'Nova lista';

  @override
  String get new_password => 'Nova senha';

  @override
  String get new_password_error => 'Error setting new password';

  @override
  String get new_password_success => 'The new password has been set';

  @override
  String get new_password_text => 'Set a new password';

  @override
  String get new_token_generated => 'New API Token generated';

  @override
  String get new_trail => 'Nova trilha';

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
  String get no_account => 'Não tem uma conta?';

  @override
  String get no_api_tokens => 'You have no API Tokens';

  @override
  String get no_comments_so_far => 'No comments so far';

  @override
  String get no_data => 'Sem dados';

  @override
  String get no_description_for_now => 'No description for now';

  @override
  String get no_gps_data_in_image => 'No GPS data in image';

  @override
  String get no_grid => 'No Grid';

  @override
  String get no_notifications => 'No notifications';

  @override
  String get no_photos_here => 'No photos here';

  @override
  String get no_preference => 'Nenhuma preferência';

  @override
  String get no_results => 'Nenhum resultado encontrado';

  @override
  String get no_routes_added => 'No routes added';

  @override
  String get no_waypoints_yet => 'No waypoints yet';

  @override
  String get norwegian => 'Norwegian';

  @override
  String get not_a_valid_email_address => 'Não um endereço de e-mail válido';

  @override
  String get not_a_valid_url => 'Not a valid URL';

  @override
  String get not_completed => 'Não preenchido';

  @override
  String notification_comment_mention(Object user) {
    return '$user mentioned you in a comment';
  }

  @override
  String notification_list_create(Object user) {
    return '$user created a new list';
  }

  @override
  String notification_list_share(Object user) {
    return '$user shared a list with you';
  }

  @override
  String get notification_new_follower => 'You have a new follower';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user created a summit log on your trail \"$trail\"';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user mentioned you in a summit log';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user left a comment on your trail \"$trail\"';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user created a new trail';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user liked your trail \"$trail\"';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user mentioned you in a trail';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user shared a trail with you';
  }

  @override
  String get notifications => 'Notifications';

  @override
  String object_share_error(Object object) {
    return 'A $object must be public to be shared across instances.';
  }

  @override
  String get off => 'Desligar';

  @override
  String get only_me => 'Only me';

  @override
  String get open_in_new_tab => 'Open in new tab';

  @override
  String get or => 'ou';

  @override
  String get orientation => 'Orientação';

  @override
  String get paper_size => 'Tamanho do papel';

  @override
  String get paragraph => 'Paragraph';

  @override
  String get parking => 'Parking';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Senha';

  @override
  String get password_confirm => 'Confirm password';

  @override
  String get password_reset_sent => 'A password reset email has been sent';

  @override
  String get password_reset_text =>
      'We will send a reset link to your email address.';

  @override
  String get password_updated => 'Pasword atualizada';

  @override
  String get passwords_must_match => 'Passwords must match';

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
  String get pick_a_trail => 'Escolha uma trilha';

  @override
  String get planned_a_trail => 'planned a trail';

  @override
  String get planned_tours => 'Planned tours';

  @override
  String get pois => 'POIs';

  @override
  String get polish => 'Polonês';

  @override
  String get portuguese => 'Português';

  @override
  String get print => 'Imprimir';

  @override
  String get privacy => 'Privacy';

  @override
  String get private => 'Private';

  @override
  String get profile => 'Perfil';

  @override
  String get public => 'Público';

  @override
  String get public_access => 'Public access';

  @override
  String get public_share_everyone =>
      'Everyone on the internet with the link can see this trail';

  @override
  String get public_share_limited =>
      'Only people with access can open the link';

  @override
  String get public_transport => 'Public transport';

  @override
  String get radius => 'Raio';

  @override
  String get railway_station => 'Railway station';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get read_more => 'Ler mais';

  @override
  String get ready_to_join => 'Ready to join';

  @override
  String get recalculate_elevation_data => 'Recalculate elevation data';

  @override
  String get recalculating_elevation_data_hint =>
      'Recalculating elevation data will erase the existing elevation data, if any, and replace it with data from Valhalla.';

  @override
  String get register => 'Registo';

  @override
  String get remote_users_cannot_edit => 'Remote users cannot edit';

  @override
  String get removed_trail_from => 'Trilha removida de';

  @override
  String get removed_trails_from => 'Trilhos removidos de';

  @override
  String get required => 'Obrigatório';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Reset';

  @override
  String get reset_password => 'Reset Password';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => 'Retomar a gravação?';

  @override
  String get reverse_direction => 'Reverse direction';

  @override
  String get road => 'Road';

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
  String get route_point => 'Route Point';

  @override
  String get russian => 'Russian';

  @override
  String get save => 'Guardar';

  @override
  String get save_list => 'Gravar lista';

  @override
  String get save_track => 'Save track';

  @override
  String get save_trail => 'Guardar trilho';

  @override
  String get save_your_trail_first => 'Salve sua trilha primeiro';

  @override
  String get search => 'Search';

  @override
  String get search_cities => 'Procurar cidades';

  @override
  String get search_for_trails_places => 'Procurar trilhos, locais';

  @override
  String get search_list => 'Search list';

  @override
  String get search_places => 'Search places';

  @override
  String get search_trails => 'Procurar trilhos';

  @override
  String get select_date => 'Select date';

  @override
  String get select_list => 'Selecionar lista';

  @override
  String get selected => 'selected';

  @override
  String get send_to => 'Send to...';

  @override
  String get set_private => 'Set private';

  @override
  String get set_public => 'Set public';

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
  String get settings_saved => 'Settings saved';

  @override
  String get share => 'Partilhar';

  @override
  String get share_profile => 'Share profile';

  @override
  String get share_this_list => 'Partilhar esta lista';

  @override
  String get share_this_trail => 'Partilhar este percurso';

  @override
  String get shared => 'Shared';

  @override
  String get shared_by => 'Partilhado por';

  @override
  String get shared_with => 'Partilhado com';

  @override
  String get shelter => 'Shelter';

  @override
  String get shortest => 'shortest';

  @override
  String get show_in_overview => 'Mostrar na vista geral';

  @override
  String get show_less => 'Show less';

  @override
  String get show_on_map => 'Mostrar no mapa';

  @override
  String get shower => 'Shower';

  @override
  String get skiing => 'Skiing';

  @override
  String get slogan => 'Guarde as suas aventuras!';

  @override
  String get slope => 'Inclinação';

  @override
  String get someone => 'Someone';

  @override
  String get sort => 'Ordenar';

  @override
  String get spanish => 'Espanhol';

  @override
  String get speed => 'Velocidade';

  @override
  String get start => 'Start';

  @override
  String get statistics => 'Statistics';

  @override
  String get stop_drawing => 'Parar desenho';

  @override
  String get stop_editing => 'Stop editing';

  @override
  String get strava_integration_after_date_hint =>
      'If your account has a large amount of acitivities you may run into Strava\'s API rate limit preventing you from syncing all activities at once. To mitigate this issue you can set an \"After\" date below so that only activities that were recorded after this date are synced.';

  @override
  String get subcategories => 'Subcategorias';

  @override
  String get subway_stop => 'Subway entrance';

  @override
  String get summit => 'Summit';

  @override
  String get summit_book => 'Livro da cimeira';

  @override
  String get table => 'Tabela';

  @override
  String get tags => 'Tags';

  @override
  String get text => 'Texto';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get tilesets => 'Camada de renderização personalizada';

  @override
  String get time => 'Time';

  @override
  String get toilets => 'Toilets';

  @override
  String get top_speed => 'Top Speed';

  @override
  String get tourism => 'Tourism';

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
  String get trail_copied_successfully => 'trail copied successfully';

  @override
  String get trail_has_no_gpx => 'This trail has no GPX data.';

  @override
  String get trail_not_in_list => 'Trail is not in any list';

  @override
  String get trail_not_shared => 'Não partilhado com ninguém';

  @override
  String get trail_saved_successfully => 'Percurso gravado com sucesso';

  @override
  String get some_waypoints_failed_to_save =>
      'Trail saved, but some waypoints failed to save';

  @override
  String get trails_for_you => 'Trilhos para si';

  @override
  String get tram_stop => 'Tram stop';

  @override
  String get unchanged => 'unchanged';

  @override
  String get units => 'Unidades';

  @override
  String get unlink => 'Unlink';

  @override
  String get upload_file => 'Subir arquivo';

  @override
  String get upload_gpx => 'Carregar GPX';

  @override
  String get upload_new_file => 'Upload new file';

  @override
  String get uploaded => 'carregado';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Successfully uploaded trail to Hammerhead';

  @override
  String get use_hills => 'Use hills';

  @override
  String get use_roads => 'Use Roads';

  @override
  String get users => 'Users';

  @override
  String get username => 'Nome de utilizador';

  @override
  String get username_not_unique =>
      'This username is already taken. Please try another.';

  @override
  String get view => 'Ver';

  @override
  String get viewpoint => 'Viewpoint';

  @override
  String get visibilty => 'Visibility';

  @override
  String get visibilty_status => 'Visibility status';

  @override
  String get walking_speed => 'Walking speed';

  @override
  String get water => 'Water';

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
  String get width => 'Width';

  @override
  String get wrong_username_or_password =>
      'Nome de utilizador ou palavra-passe errados';

  @override
  String get you_can => 'Podes';

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
